local _M = {}
local redis = require "resty.redis"
local keydb_config = require "cache.keydb_config"
local resolver_utils = require "utils.resolver_utils"

-- Helper function to create and configure Redis connection
local function create_keydb_connection()
    local red = redis:new()
    
    -- Set timeouts
    red:set_timeouts(
        keydb_config.get_connect_timeout(),
        keydb_config.get_send_timeout(),
        keydb_config.get_read_timeout()
    )
    
    -- Parse KeyDB URL
    local keydb_url = keydb_config.get_keydb_url()
    local url_parts, err = resolver_utils.parse_url(keydb_url)
    
    if not url_parts then
        ngx.log(ngx.ERR, "keydb_l3_cache: Invalid KeyDB URL format: ", keydb_url, " Error: ", err or "unknown")
        return nil, "Invalid URL format"
    end
    
    local host = url_parts.host
    local port = tonumber(url_parts.port)
    
    -- Connect to KeyDB
    local ok, err = red:connect(host, port)
    if not ok then
        ngx.log(ngx.ERR, "keydb_l3_cache: Failed to connect to KeyDB: ", err)
        return nil, err
    end
    
    return red, nil
end

-- Helper function to close connection with keepalive
local function close_keydb_connection(red)
    if not red then
        return
    end
    
    -- Put connection into keepalive pool
    local ok, err = red:set_keepalive(
        keydb_config.get_max_idle_timeout(),
        keydb_config.get_pool_size()
    )
    
    if not ok then
        ngx.log(ngx.WARN, "keydb_l3_cache: Failed to set keepalive: ", err)
    end
end

-- Helper function to validate data for storage
local function validate_data_for_storage(data)
    if type(data) ~= "string" then
        ngx.log(ngx.ERR, "keydb_l3_cache: Data must be a string, got: ", type(data))
        return nil
    end
    return data
end

-- Set data in KeyDB L3 cache
function _M.set(key, value, ttl)
    if not key or not value then
        ngx.log(ngx.ERR, "keydb_l3_cache: Key and value are required")
        return false, "Key and value are required"
    end
    
    ttl = ttl or keydb_config.get_default_ttl() or 3600
    local max_ttl = keydb_config.get_max_ttl() or 86400
    ttl = math.min(ttl, max_ttl)
    
    local red, err = create_keydb_connection()
    if not red then
        return false, err
    end
    
    local cache_key = key
    local validated_data = validate_data_for_storage(value)
    if not validated_data then
        close_keydb_connection(red)
        return false, "Invalid data format"
    end
    
    -- Set data with TTL
    local ok, err = red:setex(cache_key, ttl, validated_data)
    
    close_keydb_connection(red)
    
    if not ok then
        ngx.log(ngx.ERR, "keydb_l3_cache: Failed to set cache key ", cache_key, ": ", err)
        return false, err
    end
    
    ngx.log(ngx.DEBUG, "keydb_l3_cache: Set cache key ", cache_key, " with TTL ", ttl)
    return true, nil
end

-- Get data from KeyDB L3 cache
function _M.get(key)
    if not key then
        ngx.log(ngx.ERR, "keydb_l3_cache: Key is required")
        return nil, "Key is required"
    end
    
    local red, err = create_keydb_connection()
    if not red then
        return nil, err
    end
    
    local cache_key = key
    
    -- Get data from cache
    local data, err = red:get(cache_key)
    
    close_keydb_connection(red)
    
    if not data then
        ngx.log(ngx.ERR, "keydb_l3_cache: Failed to get cache key ", cache_key, ": ", err)
        return nil, err
    end
    
    if data == ngx.null then
        ngx.log(ngx.DEBUG, "keydb_l3_cache: Cache miss for key ", cache_key)
        return nil, "cache miss"
    end
    
    ngx.log(ngx.DEBUG, "keydb_l3_cache: Cache hit for key ", cache_key)
    return data, nil
end

-- Check if L3 cache is enabled
function _M.enabled()
    return keydb_config.enabled()
end

return _M 