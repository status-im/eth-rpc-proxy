-- auth_config.lua - Configuration module for authentication settings
local _M = {}

-- Initialize configuration values once at worker startup
function _M.init()
    -- JWT rate limiting configuration
    _M.requests_per_token = tonumber(os.getenv("REQUESTS_PER_TOKEN")) or 100
    
    -- JWT token expiry configuration
    _M.token_expiry_minutes = tonumber(os.getenv("TOKEN_EXPIRY_MINUTES")) or 10
    
    -- Log the initialized values
    ngx.log(ngx.NOTICE, "auth_config: Initialized REQUESTS_PER_TOKEN = ", _M.requests_per_token)
    ngx.log(ngx.NOTICE, "auth_config: Initialized TOKEN_EXPIRY_MINUTES = ", _M.token_expiry_minutes)
    
    -- You can add other auth-related config here
    -- _M.jwt_secret = os.getenv("JWT_SECRET") or "default-secret"
end

-- Getter functions for clean access
function _M.get_requests_per_token()
    return _M.requests_per_token
end

function _M.get_token_expiry_minutes()
    return _M.token_expiry_minutes
end

return _M 