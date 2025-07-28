local _M = {}
local yaml = require("lyaml")

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

return _M 