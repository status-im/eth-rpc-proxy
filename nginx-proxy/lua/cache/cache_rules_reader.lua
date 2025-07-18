local _M = {}
local yaml = require("lyaml")

local config = nil
local config_loaded = false
local config_file_path = nil

function _M.load_config(premature, file_path)
    if premature then
        return
    end
    
    config_file_path = file_path
    ngx.log(ngx.INFO, "cache_rules_reader: Loading config from: ", config_path)
    
    local config_data = _M.read_yaml_config(config_path)
    
    if config_data then
        if _M.validate_config(config_data) then
            config = config_data
            config_loaded = true
            ngx.log(ngx.NOTICE, "cache_rules_reader: Config loaded successfully from: ", config_path)
        else
            ngx.log(ngx.ERR, "cache_rules_reader: Config validation failed for: ", config_path)
            config_loaded = false
        end
    else
        ngx.log(ngx.ERR, "cache_rules_reader: Failed to read config from: ", config_path)
        config_loaded = false
    end
end

function _M.init(file_path)
    config_file_path = file_path or os.getenv("CACHE_RULES_FILE") or "/app/cache_rules.yaml"
    ngx.log(ngx.INFO, "cache_rules_reader: Initializing with config file: ", config_file_path)
    
    local ok, err = ngx.timer.at(0, _M.load_config, config_file_path)
    if not ok then
        ngx.log(ngx.ERR, "cache_rules_reader: Failed to create initial timer: ", err)
        return false
    end
    
    return true
end


function _M.read_yaml_config(file_path)
    local file = io.open(file_path, "r")
    if not file then
        ngx.log(ngx.ERR, "cache_rules_reader: Could not open config file: ", file_path)
        return nil
    end
    
    local content = file:read("*all")
    file:close()
    
    if not content or content == "" then
        ngx.log(ngx.ERR, "cache_rules_reader: Config file is empty: ", file_path)
        return nil
    end
    
    local ok, config_data = pcall(yaml.load, content)
    if not ok then
        ngx.log(ngx.ERR, "cache_rules_reader: Failed to parse YAML config: ", config_data)
        return nil
    end
    
    return config_data
end

function _M.validate_config(cfg)
    if not cfg then
        ngx.log(ngx.ERR, "cache_rules_reader: Config is nil")
        return false
    end
    
    if not cfg.ttl_defaults then
        ngx.log(ngx.ERR, "cache_rules_reader: Missing ttl_defaults section")
        return false
    end
    
    if not cfg.cache_rules then
        ngx.log(ngx.ERR, "cache_rules_reader: Missing cache_rules section")
        return false
    end
    
    if not cfg.ttl_defaults.default then
        ngx.log(ngx.ERR, "cache_rules_reader: Missing ttl_defaults.default section")
        return false
    end
    
    local required_ttls = {"permanent", "short", "minimal"}
    for _, ttl_name in ipairs(required_ttls) do
        if not cfg.ttl_defaults.default[ttl_name] then
            ngx.log(ngx.ERR, "cache_rules_reader: Missing default TTL constant: ", ttl_name)
            return false
        end
    end
    
    ngx.log(ngx.INFO, "cache_rules_reader: Configuration validation passed")
    return true
end

function _M.get_config()
    if not config_loaded then
        ngx.log(ngx.WARN, "cache_rules_reader: Config not loaded yet, attempting to load")
        _M.load_config(false, config_file_path)
    end
    return config
end

return _M 