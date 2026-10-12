local json = require("cjson")
local auth_config = require("auth.auth_config")
local auth_utils = require("utils.auth_utils")
local token_limit = require("auth.token_limit")

-- Extract JWT token from Authorization header or query parameters
local token, token_source = auth_utils.extract_jwt_token()

if not token then
    ngx.status = 401
    ngx.exit(401)
end

-- If token was extracted from query parameters, remove them from the request
-- since no other query params are expected, we can safely clear all args
if token_source == "query" then
    ngx.req.set_uri_args({})
end

-- Get configuration values dynamically
local requests_per_token = auth_config.get_requests_per_token()
local token_expiry_minutes = auth_config.get_token_expiry_minutes()

-- Use token as cache key
local cache_key = "jwt_valid:" .. token
local usage_key = "jwt_usage:" .. token

-- Usage counter TTL should be longer than cache TTL to prevent inconsistencies
local cache_ttl = token_expiry_minutes * 60
local usage_ttl = cache_ttl + 60

-- Counts this request and answers it by the count including it. The count
-- is taken with one atomic incr: reading it and writing it back separately
-- lets concurrent requests read the same count and all slip under the limit.
local function count_and_answer(cache_status)
    local usage, err = ngx.shared.jwt_tokens:incr(usage_key, 1, 0, usage_ttl)
    if not usage then
        ngx.log(ngx.WARN, "Failed to update usage counter for token: ", err)
        usage = 1
    end

    local answer = token_limit.decide(usage, requests_per_token, cache_status)
    if answer.status ~= 200 then
        ngx.log(ngx.WARN, "Rate limit exceeded for token: ", usage, "/", requests_per_token)
    end
    for name, value in pairs(answer.headers) do
        ngx.header[name] = value
    end
    ngx.status = answer.status
    ngx.exit(answer.status)
end

-- Check if token is in cache (previously validated by Go service)
local cached_result = ngx.shared.jwt_tokens:get(cache_key)

if cached_result then
    count_and_answer("HIT")
end

-- Cache miss - validate with Go service
-- Get current auth service URL for logging
local current_url = auth_config.get_go_auth_service_url()

-- Create subrequest to Go auth service
-- Always use Bearer token format for internal verification
local auth_header_for_go = "Bearer " .. token
local res = ngx.location.capture("/_auth_go_verify", {
    method = ngx.HTTP_GET,
    headers = {
        ["Authorization"] = auth_header_for_go
    }
})

if res.status == 200 then
    -- Token is valid: cache it for the duration of token expiry and count
    -- this request, which concurrent first requests do too
    local cache_success = ngx.shared.jwt_tokens:set(cache_key, "valid", cache_ttl)
    if not cache_success then
        ngx.log(ngx.WARN, "Failed to cache valid JWT token")
    end

    count_and_answer("MISS")
    
elseif res.status == 429 then
    -- Rate limit exceeded at Go service level
    ngx.log(ngx.WARN, "Rate limit exceeded at Go service")
    ngx.header["X-RateLimit-Limit"] = tostring(requests_per_token)
    ngx.header["X-RateLimit-Remaining"] = "0"
    ngx.header["X-Cache-Status"] = "MISS"
    ngx.status = 401
    ngx.exit(401)
    
else
    -- Token is invalid
    ngx.log(ngx.WARN, "JWT validation failed at Go service: ", res.status)
    ngx.header["X-Cache-Status"] = "MISS"
    ngx.status = 401
    ngx.exit(401)
end 