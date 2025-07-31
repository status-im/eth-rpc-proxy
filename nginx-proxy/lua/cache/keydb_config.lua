local _M = {}
local lyaml = require "lyaml"
local resolver_utils = require "utils.resolver_utils"

-- Helper function to set default configuration values
local function set_default_values()
    ngx.log(ngx.NOTICE, "keydb_config: set_default_values() - Setting default configuration values")
    
    _M.connect_timeout = 100
    _M.send_timeout = 1000
    _M.read_timeout = 1000
    _M.pool_size = 10
    _M.max_idle_timeout = 10000
    _M.default_ttl = 3600
    _M.max_ttl = 86400
    _M._enabled = true  -- Default: KeyDB L3 cache is enabled (internal variable)
    
    ngx.log(ngx.NOTICE, "keydb_config: set_default_values() - Default values set: ",
            "connect_timeout=", _M.connect_timeout, ", send_timeout=", _M.send_timeout,
            ", read_timeout=", _M.read_timeout, ", pool_size=", _M.pool_size,
            ", max_idle_timeout=", _M.max_idle_timeout, ", default_ttl=", _M.default_ttl,
            ", max_ttl=", _M.max_ttl, ", enabled=", tostring(_M._enabled))
end

function _M.load_config(premature)
    if premature then
        ngx.log(ngx.NOTICE, "keydb_config: load_config called prematurely, skipping")
        return
    end
    
    ngx.log(ngx.NOTICE, "keydb_config: Starting configuration loading process")
    
    local config_path = os.getenv("KEYDB_CONFIG_FILE") or "/app/keydb_config.yaml"
    local base_keydb_url = os.getenv("KEYDB_URL") or "redis://keydb:6379"
    local custom_dns = os.getenv("CUSTOM_DNS") or "127.0.0.11"  -- Docker DNS
    
    ngx.log(ngx.NOTICE, "keydb_config: Environment variables - CONFIG_FILE: ", config_path, 
            ", KEYDB_URL: ", base_keydb_url, ", CUSTOM_DNS: ", custom_dns)
    
    ngx.log(ngx.NOTICE, "keydb_config: Starting URL resolution for: ", base_keydb_url, " with DNS: ", custom_dns)
    local resolved_url, err = resolver_utils.resolve_url_with_custom_dns(base_keydb_url, custom_dns)
    
    local final_url = (resolved_url or base_keydb_url):gsub("/$", "")
    _M.keydb_url = final_url
    
    if resolved_url then
        ngx.log(ngx.NOTICE, "keydb_config: Successfully resolved KeyDB URL from '", base_keydb_url, "' to: ", _M.keydb_url)
    else
        ngx.log(ngx.WARN, "keydb_config: Failed to resolve KeyDB URL '", base_keydb_url, "', using original: ", _M.keydb_url, " Error: ", err or "unknown")
    end

    -- Read and parse YAML config file
    ngx.log(ngx.NOTICE, "keydb_config: Attempting to read YAML config from: ", config_path)
    local config_data = _M.read_yaml_config(config_path)
    
    if config_data then
        ngx.log(ngx.NOTICE, "keydb_config: Successfully parsed YAML config, applying settings")
        
        _M.connect_timeout = config_data.connection and config_data.connection.connect_timeout or 100
        _M.send_timeout = config_data.connection and config_data.connection.send_timeout or 1000
        _M.read_timeout = config_data.connection and config_data.connection.read_timeout or 1000
        
        ngx.log(ngx.NOTICE, "keydb_config: Connection timeouts set - connect: ", _M.connect_timeout, 
                ", send: ", _M.send_timeout, ", read: ", _M.read_timeout)
        
        _M.pool_size = config_data.keepalive and config_data.keepalive.pool_size or 10
        _M.max_idle_timeout = config_data.keepalive and config_data.keepalive.max_idle_timeout or 10000
        
        ngx.log(ngx.NOTICE, "keydb_config: Keepalive settings set - pool_size: ", _M.pool_size, 
                ", max_idle_timeout: ", _M.max_idle_timeout)
        
        _M.default_ttl = config_data.cache and config_data.cache.default_ttl or 3600
        _M.max_ttl = config_data.cache and config_data.cache.max_ttl or 86400
        
        ngx.log(ngx.NOTICE, "keydb_config: Cache TTL settings set - default: ", _M.default_ttl, 
                ", max: ", _M.max_ttl)
        
        -- Load enabled flag, default to true if not specified
        _M._enabled = true
        if config_data.enabled ~= nil then
            _M._enabled = config_data.enabled
            ngx.log(ngx.NOTICE, "keydb_config: L3 cache enabled flag explicitly set to: ", tostring(_M._enabled))
        else
            ngx.log(ngx.NOTICE, "keydb_config: L3 cache enabled flag not specified, defaulting to: ", tostring(_M._enabled))
        end
        
        ngx.log(ngx.NOTICE, "keydb_config: Configuration successfully loaded from ", config_path, 
                " (L3 cache enabled: ", tostring(_M._enabled), ")")
    else
        -- Fallback to default values if YAML config fails
        ngx.log(ngx.WARN, "keydb_config: YAML config loading failed, falling back to default values")
        set_default_values()
        ngx.log(ngx.NOTICE, "keydb_config: Default values applied - L3 cache enabled: ", tostring(_M._enabled))
    end
end

-- Initialize configuration using timer
function _M.init()
    ngx.log(ngx.NOTICE, "keydb_config: Starting initialization process")
    
    -- Set default values immediately to prevent race conditions
    local default_url = os.getenv("KEYDB_URL") or "redis://keydb:6379"
    _M.keydb_url = default_url:gsub("/$", "")
    
    ngx.log(ngx.NOTICE, "keydb_config: Set initial KeyDB URL to: ", _M.keydb_url)
    
    set_default_values()
    ngx.log(ngx.NOTICE, "keydb_config: Applied default configuration values as fallback")
    
    -- Schedule config loading using timer
    ngx.log(ngx.NOTICE, "keydb_config: Scheduling configuration loading via timer")
    local ok, err = ngx.timer.at(0, _M.load_config)
    if not ok then
        ngx.log(ngx.ERR, "keydb_config: Failed to create timer for config loading: ", err)
    else
        ngx.log(ngx.NOTICE, "keydb_config: Timer successfully created for config loading")
    end
    
    ngx.log(ngx.NOTICE, "keydb_config: Initialization process completed")
end

-- Read and parse YAML configuration file
function _M.read_yaml_config(file_path)
    ngx.log(ngx.NOTICE, "keydb_config: read_yaml_config() - Attempting to open file: ", file_path)
    
    local file = io.open(file_path, "r")
    if not file then
        ngx.log(ngx.ERR, "keydb_config: read_yaml_config() - Could not open config file: ", file_path)
        return nil
    end
    
    ngx.log(ngx.NOTICE, "keydb_config: read_yaml_config() - File opened successfully, reading content")
    
    local content = file:read("*all")
    file:close()
    
    if not content or content == "" then
        ngx.log(ngx.ERR, "keydb_config: read_yaml_config() - Config file is empty: ", file_path)
        return nil
    end
    
    ngx.log(ngx.NOTICE, "keydb_config: read_yaml_config() - Content read successfully (", string.len(content), " bytes), parsing YAML")
    
    local ok, config_data = pcall(lyaml.load, content)
    if not ok then
        ngx.log(ngx.ERR, "keydb_config: read_yaml_config() - Failed to parse YAML config: ", config_data)
        return nil
    end
    
    ngx.log(ngx.NOTICE, "keydb_config: read_yaml_config() - YAML parsed successfully")
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