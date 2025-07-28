local json = require("cjson")
local cache_rules_reader = require("cache.cache_rules_reader")

local _M = {}

-- Module state
local config = nil
local config_loaded = false

function _M.load_config(premature, file_path)
    if premature then
        return
    end
    
    ngx.log(ngx.INFO, "cache_rules: Loading config from: ", file_path)
    
    local config_data = cache_rules_reader.read_yaml_config(file_path)
    
    if config_data then
        if cache_rules_reader.validate_config(config_data) then
            config = config_data
            config_loaded = true
            ngx.log(ngx.NOTICE, "cache_rules: Config loaded successfully from: ", file_path)
        else
            ngx.log(ngx.ERR, "cache_rules: Config validation failed for: ", file_path)
            config_loaded = false
        end
    else
        ngx.log(ngx.ERR, "cache_rules: Failed to read config from: ", file_path)
        config_loaded = false
    end
end

function _M.init(file_path)
    ngx.log(ngx.INFO, "cache_rules: Initializing with config file: ", file_path)
    
    local ok, err = ngx.timer.at(0, _M.load_config, file_path)
    if not ok then
        ngx.log(ngx.ERR, "cache_rules: Failed to create initial timer: ", err)
        return false
    end
    
    return true
end

function _M.classify_method_cache_type(method, params, chain, network)
    if not config_loaded or not config or not config.cache_rules then
        return nil
    end
    
    local method_rules = config.cache_rules[method]
    if not method_rules then
        return nil
    end
    
    if type(method_rules) == "string" then
        return method_rules
    end

    return nil
end

function _M.get_ttl_for_cache_type(cache_type, chain, network)
    if not config_loaded or not config or not config.ttl_defaults then
        local fallback_ttls = {
            permanent = 86400,
            short = 5,
            minimal = 0
        }
        return fallback_ttls[cache_type] or 0
    end
    
    local network_key = chain .. ":" .. network
    local chain_key = chain
    
    -- Try network-specific config first
    if config.ttl_defaults[network_key] and config.ttl_defaults[network_key][cache_type] ~= nil then
        return config.ttl_defaults[network_key][cache_type]
    end
    
    -- Try chain-specific config
    if config.ttl_defaults[chain_key] and config.ttl_defaults[chain_key][cache_type] ~= nil then
        return config.ttl_defaults[chain_key][cache_type]
    end
    
    -- Fall back to default config
    if config.ttl_defaults.default and config.ttl_defaults.default[cache_type] ~= nil then
        return config.ttl_defaults.default[cache_type]
    end
    
    return 0
end

-- Main function to get cache information
function _M.get_cache_info(chain, network, decoded_body)
    if not decoded_body or not decoded_body.method then
        return {
            cache_type = "none",
            ttl = 0
        }
    end
    
    local method = decoded_body.method
    local params = decoded_body.params or {}
    
    -- Classify method
    local cache_type = _M.classify_method_cache_type(method, params, chain, network)
    
    if not cache_type or cache_type == "none" then
        return {
            cache_type = "none",
            ttl = 0
        }
    end
    
    -- Get TTL
    local ttl = _M.get_ttl_for_cache_type(cache_type, chain, network)
    
    -- If TTL is 0, treat as non-cacheable
    if ttl == 0 then
        return {
            cache_type = "none",
            ttl = 0
        }
    end
    
    return {
        cache_type = cache_type,
        ttl = ttl
    }
end

return _M 