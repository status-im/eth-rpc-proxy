local _M = {}
local lyaml = require "lyaml"
local resolver_utils = require "utils.resolver_utils"

-- Helper function to set default configuration values
local function set_default_values()
    _M.connect_timeout = 100
    _M.send_timeout = 1000
    _M.read_timeout = 1000
    _M.pool_size = 10
    _M.max_idle_timeout = 10000
    _M.default_ttl = 3600
    _M.max_ttl = 86400
    _M._enabled = true  -- Default: KeyDB L3 cache is enabled (internal variable)
end

function _M.load_config(premature)
    if premature then
        return
    end
    
    local config_path = os.getenv("KEYDB_CONFIG_FILE") or "/app/keydb_config.yaml"
    local base_keydb_url = os.getenv("KEYDB_URL") or "redis://keydb:6379"
    local custom_dns = os.getenv("CUSTOM_DNS") or "127.0.0.11"  -- Docker DNS
    local resolved_url, err = resolver_utils.resolve_url_with_custom_dns(base_keydb_url, custom_dns)
    local final_url = (resolved_url or base_keydb_url):gsub("/$", "")
    _M.keydb_url = final_url
    
    if resolved_url then
        ngx.log(ngx.NOTICE, "keydb_config: Resolved KeyDB URL: ", _M.keydb_url)
    else
        ngx.log(ngx.WARN, "keydb_config: Failed to resolve KeyDB URL, using original: ", _M.keydb_url, " Error: ", err or "unknown")
    end

    -- Read and parse YAML config file
    local config_data = _M.read_yaml_config(config_path)
    
    if config_data then
        _M.connect_timeout = config_data.connection and config_data.connection.connect_timeout or 100
        _M.send_timeout = config_data.connection and config_data.connection.send_timeout or 1000
        _M.read_timeout = config_data.connection and config_data.connection.read_timeout or 1000
        
        _M.pool_size = config_data.keepalive and config_data.keepalive.pool_size or 10
        _M.max_idle_timeout = config_data.keepalive and config_data.keepalive.max_idle_timeout or 10000
        
        _M.default_ttl = config_data.cache and config_data.cache.default_ttl or 3600
        _M.max_ttl = config_data.cache and config_data.cache.max_ttl or 86400
        
        -- Load enabled flag, default to true if not specified
        _M._enabled = true
        if config_data.enabled ~= nil then
            _M._enabled = config_data.enabled
        end
        
        ngx.log(ngx.NOTICE, "keydb_config: Loaded from ", config_path, " (L3 cache enabled: ", tostring(_M._enabled), ")")
    else
        -- Fallback to default values if YAML config fails
        set_default_values()
        ngx.log(ngx.WARN, "keydb_config: Using default values due to config loading failure")
    end
end

-- Initialize configuration using timer
function _M.init()
    -- Set default values immediately to prevent race conditions
    local default_url = os.getenv("KEYDB_URL") or "redis://keydb:6379"
    _M.keydb_url = default_url:gsub("/$", "")
    set_default_values()
    
    -- Schedule config loading using timer
    local ok, err = ngx.timer.at(0, _M.load_config)
    if not ok then
        ngx.log(ngx.ERR, "keydb_config: Failed to create timer: ", err)
    end
end

-- Read and parse YAML configuration file
function _M.read_yaml_config(file_path)
    local file = io.open(file_path, "r")
    if not file then
        ngx.log(ngx.ERR, "keydb_config: Could not open config file: ", file_path)
        return nil
    end
    
    local content = file:read("*all")
    file:close()
    
    if not content or content == "" then
        ngx.log(ngx.ERR, "keydb_config: Config file is empty: ", file_path)
        return nil
    end
    
    local ok, config_data = pcall(lyaml.load, content)
    if not ok then
        ngx.log(ngx.ERR, "keydb_config: Failed to parse YAML config: ", config_data)
        return nil
    end
    
    return config_data
end

-- Getter functions for clean access
function _M.get_keydb_url()
    return _M.keydb_url
end

function _M.get_connect_timeout()
    return _M.connect_timeout
end

function _M.get_send_timeout()
    return _M.send_timeout
end

function _M.get_read_timeout()
    return _M.read_timeout
end

function _M.get_pool_size()
    return _M.pool_size
end

function _M.get_max_idle_timeout()
    return _M.max_idle_timeout
end

function _M.get_default_ttl()
    return _M.default_ttl
end

function _M.get_max_ttl()
    return _M.max_ttl
end

function _M.enabled()
    return _M._enabled
end

return _M 