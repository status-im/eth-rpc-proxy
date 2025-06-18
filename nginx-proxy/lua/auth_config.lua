-- auth_config.lua - Configuration module for authentication settings
local _M = {}
local cjson = require "cjson"
local resolver_utils = require "resolver_utils"

-- Initialize configuration values once at worker startup
function _M.init()
    -- Get config file path from environment variable
    local config_file_path = os.getenv("AUTH_CONFIG_FILE") or "/app/config.json"
    
    -- Get Go Auth Service URL from environment variable
    local base_auth_url = os.getenv("GO_AUTH_SERVICE_URL") or "http://auth-service:8081"
    
    -- Try to resolve auth-service URL using custom DNS if available
    local custom_dns = os.getenv("CUSTOM_DNS") or "127.0.0.11"  -- Docker DNS
    local resolved_url, err = resolver_utils.resolve_url_with_custom_dns(base_auth_url, custom_dns)
    
    if resolved_url then
        _M.go_auth_service_url = resolved_url
        ngx.log(ngx.NOTICE, "auth_config: Resolved auth service URL: ", resolved_url)
    else
        _M.go_auth_service_url = base_auth_url
        ngx.log(ngx.WARN, "auth_config: Failed to resolve auth service URL, using original: ", base_auth_url, " Error: ", err or "unknown")
    end
    
    -- Read and parse JSON config file
    local config_data = _M.read_json_config(config_file_path)
    
    if config_data then
        -- JWT rate limiting configuration
        _M.requests_per_token = config_data.requests_per_token or 100
        
        -- JWT token expiry configuration  
        _M.token_expiry_minutes = config_data.token_expiry_minutes or 10
        
        -- Log the initialized values
        ngx.log(ngx.NOTICE, "auth_config: Loaded from ", config_file_path)
        ngx.log(ngx.NOTICE, "auth_config: go_auth_service_url = ", _M.go_auth_service_url)
        ngx.log(ngx.NOTICE, "auth_config: base_auth_url = ", base_auth_url)
        ngx.log(ngx.NOTICE, "auth_config: requests_per_token = ", _M.requests_per_token)
        ngx.log(ngx.NOTICE, "auth_config: token_expiry_minutes = ", _M.token_expiry_minutes)
    else
        -- Fallback to environment variables if JSON config fails
        _M.requests_per_token = tonumber(os.getenv("REQUESTS_PER_TOKEN")) or 100
        _M.token_expiry_minutes = tonumber(os.getenv("TOKEN_EXPIRY_MINUTES")) or 10
        
        ngx.log(ngx.WARN, "auth_config: Failed to load JSON config, using environment variables")
        ngx.log(ngx.NOTICE, "auth_config: go_auth_service_url = ", _M.go_auth_service_url)
        ngx.log(ngx.NOTICE, "auth_config: base_auth_url = ", base_auth_url)
        ngx.log(ngx.NOTICE, "auth_config: requests_per_token = ", _M.requests_per_token)
        ngx.log(ngx.NOTICE, "auth_config: token_expiry_minutes = ", _M.token_expiry_minutes)
    end
end

-- Read and parse JSON configuration file
function _M.read_json_config(file_path)
    local file = io.open(file_path, "r")
    if not file then
        ngx.log(ngx.ERR, "auth_config: Could not open config file: ", file_path)
        return nil
    end
    
    local content = file:read("*all")
    file:close()
    
    if not content or content == "" then
        ngx.log(ngx.ERR, "auth_config: Config file is empty: ", file_path)
        return nil
    end
    
    local ok, config_data = pcall(cjson.decode, content)
    if not ok then
        ngx.log(ngx.ERR, "auth_config: Failed to parse JSON config: ", config_data)
        return nil
    end
    
    return config_data
end

-- Getter functions for clean access
function _M.get_requests_per_token()
    return _M.requests_per_token
end

function _M.get_token_expiry_minutes()
    return _M.token_expiry_minutes
end

function _M.get_go_auth_service_url()
    return _M.go_auth_service_url
end

return _M 