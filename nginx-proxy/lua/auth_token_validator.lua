local json = require("cjson")
local auth_config = require("auth_config")

-- Debug logging
ngx.log(ngx.DEBUG, "auth_token_validator: Starting JWT validation")

-- Extract Authorization header
local auth_header = ngx.var.http_authorization
ngx.log(ngx.DEBUG, "auth_token_validator: Authorization header = ", auth_header or "nil")

if not auth_header then
    ngx.log(ngx.INFO, "No Authorization header found")
    ngx.status = 401
    ngx.exit(401)
end

-- Check if it's a Bearer token
local auth_type, token = auth_header:match("^(%S+)%s+(.+)$")
if auth_type ~= "Bearer" then
    ngx.log(ngx.INFO, "Not a Bearer token: ", auth_type or "none")
    ngx.status = 401
    ngx.exit(401)
end

-- Get configuration values dynamically
local requests_per_token = auth_config.get_requests_per_token()
local token_expiry_minutes = auth_config.get_token_expiry_minutes()

-- Use token as cache key
local cache_key = "jwt_valid:" .. token
local usage_key = "jwt_usage:" .. token

-- Check if token is in cache (previously validated by Go service)
local cached_result = ngx.shared.jwt_tokens:get(cache_key)

if cached_result then
    -- Token is cached as valid, now check and increment usage
    ngx.log(ngx.DEBUG, "JWT token found in cache")
    
    -- Get current usage count
    local current_usage = ngx.shared.jwt_tokens:get(usage_key) or 0
    
    -- Check if limit exceeded
    if current_usage >= requests_per_token then
        ngx.log(ngx.WARN, "Rate limit exceeded for cached token: ", current_usage, "/", requests_per_token)
        ngx.header["X-RateLimit-Limit"] = tostring(requests_per_token)
        ngx.header["X-RateLimit-Remaining"] = "0"
        ngx.header["X-Cache-Status"] = "HIT"
        ngx.status = 429
        ngx.exit(429)
    end
    
    -- Increment usage counter  
    local new_usage = current_usage + 1
    local usage_ttl = (token_expiry_minutes * 60) + 60  -- Token expiry + 1 minute buffer
    local success = ngx.shared.jwt_tokens:set(usage_key, new_usage, usage_ttl)
    
    if not success then
        ngx.log(ngx.WARN, "Failed to update usage counter for token")
    end
    
    -- Set rate limit headers
    ngx.header["X-RateLimit-Limit"] = tostring(requests_per_token)
    ngx.header["X-RateLimit-Remaining"] = tostring(requests_per_token - new_usage)
    ngx.header["X-Cache-Status"] = "HIT"
    
    ngx.log(ngx.DEBUG, "JWT cache hit, usage: ", new_usage, "/", requests_per_token)
    ngx.status = 200
    ngx.exit(200)
end

-- Cache miss - validate with Go service
ngx.log(ngx.DEBUG, "JWT token not in cache, validating with Go service")

-- Create subrequest to Go auth service
local res = ngx.location.capture("/_auth_go_verify", {
    method = ngx.HTTP_GET,
    headers = {
        ["Authorization"] = auth_header
    }
})

if res.status == 200 then
    -- Token is valid, cache it and initialize usage counter
    ngx.log(ngx.DEBUG, "JWT validated by Go service, caching token")
    
    -- Cache the valid token for the duration of token expiry
    local cache_ttl = token_expiry_minutes * 60  -- Convert minutes to seconds
    local cache_success = ngx.shared.jwt_tokens:set(cache_key, "valid", cache_ttl)
    if not cache_success then
        ngx.log(ngx.WARN, "Failed to cache valid JWT token")
    end
    
    -- Initialize usage counter (this request counts as first usage) 
    -- Usage counter TTL should be longer than cache TTL to prevent inconsistencies
    local usage_ttl = cache_ttl + 60  -- Extra 1 minute buffer
    local usage_success = ngx.shared.jwt_tokens:set(usage_key, 1, usage_ttl)
    if not usage_success then
        ngx.log(ngx.WARN, "Failed to initialize usage counter")
    end
    
    -- Set rate limit headers  
    ngx.header["X-RateLimit-Limit"] = tostring(requests_per_token)
    ngx.header["X-RateLimit-Remaining"] = tostring(requests_per_token - 1)
    ngx.header["X-Cache-Status"] = "MISS"
    
    ngx.log(ngx.DEBUG, "JWT validation successful, usage: 1/", requests_per_token)
    ngx.status = 200
    ngx.exit(200)
    
elseif res.status == 429 then
    -- Rate limit exceeded at Go service level
    ngx.log(ngx.WARN, "Rate limit exceeded at Go service")
    ngx.header["X-RateLimit-Limit"] = tostring(requests_per_token)
    ngx.header["X-RateLimit-Remaining"] = "0"
    ngx.header["X-Cache-Status"] = "MISS"
    ngx.status = 429
    ngx.exit(429)
    
else
    -- Token is invalid
    ngx.log(ngx.WARN, "JWT validation failed at Go service: ", res.status)
    ngx.header["X-Cache-Status"] = "MISS"
    ngx.status = 401
    ngx.exit(401)
end 