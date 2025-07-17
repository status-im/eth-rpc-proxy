local _M = {}
local cjson = require "cjson"
local resolver_utils = require "utils.resolver_utils"

-- Configuration loading function that will be called by timer
function _M.load_config(premature)
    if premature then
        return
    end
    
    -- Use provided config file path or fallback
    local config_path = os.getenv("AUTH_CONFIG_FILE") or "/app/config.json"
    
    -- Get Go Auth Service URL from environment variable
    local base_auth_url = os.getenv("GO_AUTH_SERVICE_URL") or "http://auth-service:8081"
    
    -- Try to resolve auth-service URL using custom DNS if available
    local custom_dns = os.getenv("CUSTOM_DNS") or "127.0.0.11"  -- Docker DNS
    local resolved_url, err = resolver_utils.resolve_url_with_custom_dns(base_auth_url, custom_dns)
    
    -- Remove trailing slash to fix proxy_pass behavior
    local final_url = (resolved_url or base_auth_url):gsub("/$", "")
    _M.go_auth_service_url = final_url
    
    if resolved_url then
        ngx.log(ngx.NOTICE, "auth_config: Resolved auth service URL: ", _M.go_auth_service_url)
    else
        ngx.log(ngx.WARN, "auth_config: Failed to resolve auth service URL, using original: ", _M.go_auth_service_url, " Error: ", err or "unknown")
    end

    -- Read and parse JSON config file
    local config_data = _M.read_json_config(config_path)
    
    if config_data then
        -- JWT rate limiting configuration
        _M.requests_per_token = config_data.requests_per_token or 100
        
        -- JWT token expiry configuration  
        _M.token_expiry_minutes = config_data.token_expiry_minutes or 10
        
        -- Log the initialized values
        ngx.log(ngx.NOTICE, "auth_config: Loaded from ", config_path)
    else
        -- Fallback to environment variables if JSON config fails
        _M.requests_per_token = 100
        _M.token_expiry_minutes = 10
    end
        ngx.log(ngx.NOTICE, "auth_config: go_auth_service_url = ", _M.go_auth_service_url)
        ngx.log(ngx.NOTICE, "auth_config: base_auth_url = ", base_auth_url)
        ngx.log(ngx.NOTICE, "auth_config: requests_per_token = ", _M.requests_per_token)
        ngx.log(ngx.NOTICE, "auth_config: token_expiry_minutes = ", _M.token_expiry_minutes)

end

-- Initialize configuration using timer
function _M.init()
    -- Set default values immediately to prevent race conditions
    -- Remove trailing slash to fix proxy_pass behavior
    local default_url = os.getenv("GO_AUTH_SERVICE_URL") or "http://auth-service:8081"
    _M.go_auth_service_url = default_url:gsub("/$", "")
    _M.requests_per_token = 100
    _M.token_expiry_minutes = 10
    
    -- Schedule config loading using timer
    local ok, err = ngx.timer.at(0, _M.load_config)
    if not ok then
        ngx.log(ngx.ERR, "auth_config: Failed to create timer: ", err)
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