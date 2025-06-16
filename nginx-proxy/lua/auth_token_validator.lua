local jwt_validator = require("jwt_validator")

-- Check for JWT token in Authorization header
local auth_header = ngx.var.http_authorization

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

-- Validate JWT token
local valid, payload_or_error = jwt_validator.validate_token(token)
if not valid then
    ngx.log(ngx.WARN, "JWT validation failed: ", payload_or_error)
    ngx.status = 401
    ngx.exit(401)
end

local payload = payload_or_error

-- Check usage limit
local token_id = payload.jti or payload.sub
local current_usage = jwt_validator.get_usage(token_id)
local max_requests = payload.request_limit or 100

local usage_ok, usage_error = jwt_validator.check_usage_limit(token_id, current_usage, max_requests)
if not usage_ok then
    ngx.log(ngx.WARN, "Usage limit exceeded: ", usage_error)
    ngx.status = 429  -- Too Many Requests
    ngx.header["X-RateLimit-Limit"] = tostring(max_requests)
    ngx.header["X-RateLimit-Remaining"] = "0"
    ngx.exit(429)
end

-- Increment usage counter
local new_usage = jwt_validator.increment_usage(token_id)

ngx.log(ngx.INFO, "JWT authentication successful for token: ", token_id, 
        " (usage: ", new_usage, "/", max_requests, ")")

-- Return 200 to indicate successful authentication
ngx.status = 200
ngx.exit(200) 