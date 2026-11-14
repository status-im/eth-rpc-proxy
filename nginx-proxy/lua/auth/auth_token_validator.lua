local auth_config = require("auth.auth_config")
local auth_utils = require("utils.auth_utils")
local jwt_claims = require("auth.jwt_claims")
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
local max_token_ttl = auth_config.get_token_expiry_minutes() * 60

local tokens = ngx.shared.jwt_tokens

-- A validated token is remembered under the token itself; the value is its id.
local cache_key = "jwt_valid:" .. token

-- The usage counter outlives every token it counts, so that a token never
-- finds its counter gone and starts from zero.
local COUNTER_MARGIN = 60

-- Counts this request and answers it by the count including it.
--
-- Requests are counted per token id, not per token: the auth service signs a
-- new token (same jti, later iat) every time one solved puzzle is submitted,
-- and all of them have to share one limit.
--
-- The count is taken with one atomic incr: reading it and writing it back
-- separately lets concurrent requests read the same count and all slip under
-- the limit.
local function count_and_answer(token_id, counter_ttl, cache_status)
    local usage, err = tokens:incr("jwt_usage:" .. token_id, 1, 0, counter_ttl)
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

-- Makes sure the usage counter of token_id, if there is one, lives for at least
-- ttl seconds more. A token validated later than the one that created the
-- counter may expire later, and the counter must not go before it.
local function keep_counter_for(token_id, ttl)
    local usage_key = "jwt_usage:" .. token_id
    local remaining = tokens:ttl(usage_key)
    if remaining and remaining > 0 and remaining < ttl then
        tokens:expire(usage_key, ttl)
    end
end

-- Seconds the token stays valid, by its exp claim, never longer than the
-- configured token lifetime.
local function token_ttl(claims)
    local exp = tonumber(claims.exp)
    if not exp then
        return max_token_ttl
    end
    return math.max(1, math.min(exp - ngx.time(), max_token_ttl))
end

-- Check if token is in cache (previously validated by Go service)
local cached_token_id = tokens:get(cache_key)

if cached_token_id then
    -- The counter was sized when the token was validated; the ttl here only
    -- matters if the counter was evicted.
    count_and_answer(cached_token_id, max_token_ttl + COUNTER_MARGIN, "HIT")
end

-- Cache miss - validate with Go service
-- Always use Bearer token format for internal verification
local res = ngx.location.capture("/_auth_go_verify", {
    method = ngx.HTTP_GET,
    headers = {
        ["Authorization"] = "Bearer " .. token
    }
})

if res.status == 200 then
    -- The claims are trusted from here on: the auth service checked the signature.
    local claims = jwt_claims.decode(token) or {}
    local token_id = claims.jti
    if type(token_id) ~= "string" or token_id == "" then
        token_id = token
    end

    -- Token is valid: remember it until it expires and count this request,
    -- which concurrent first requests do too
    local ttl = token_ttl(claims)
    local cache_success = tokens:set(cache_key, token_id, ttl)
    if not cache_success then
        ngx.log(ngx.WARN, "Failed to cache valid JWT token")
    end

    keep_counter_for(token_id, ttl + COUNTER_MARGIN)
    count_and_answer(token_id, ttl + COUNTER_MARGIN, "MISS")

elseif res.status == 429 then
    -- Rate limit exceeded at Go service level
    ngx.log(ngx.WARN, "Rate limit exceeded at Go service")
    ngx.header["X-RateLimit-Limit"] = tostring(requests_per_token)
    ngx.header["X-RateLimit-Remaining"] = "0"
    ngx.header["X-Cache-Status"] = "MISS"
    ngx.status = 401
    ngx.exit(401)

elseif res.status >= 500 then
    -- The auth service is down or slow, which says nothing about the token.
    -- A 401 here would make the client throw a good token away and solve a
    -- new puzzle, so answer with a server error instead.
    ngx.log(ngx.ERR, "Auth service failed to verify a token: ", res.status)
    ngx.status = 503
    ngx.exit(503)

else
    -- Token is invalid
    ngx.log(ngx.WARN, "JWT validation failed at Go service: ", res.status)
    ngx.header["X-Cache-Status"] = "MISS"
    ngx.status = 401
    ngx.exit(401)
end
