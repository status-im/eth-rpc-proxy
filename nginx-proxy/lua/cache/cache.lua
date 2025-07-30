local json = require("cjson")
local cache_rules = require("cache.cache_rules")
local mlcache = require("resty.mlcache")
local keydb_l3_cache = require("cache.keydb_l3_cache")

local _M = {}

-- Initialize mlcache instances for different cache types
local cache_instances = {}

-- Function to reset cache instances (for testing)
function _M.reset_cache_instances()
    cache_instances = {}
end

-- L3 cache callback function for mlcache
local function l3_callback(cache_key)
    -- Check if L3 cache is enabled
    if not keydb_l3_cache.enabled() then
        ngx.log(ngx.DEBUG, "L3 cache disabled, skipping for key: ", cache_key)
        return nil
    end
    
    local data, err = keydb_l3_cache.get(cache_key)
    if err and err ~= "cache miss" then
        ngx.log(ngx.ERR, "L3 cache error for key ", cache_key, ": ", err)
        return nil
    end
    
    -- Log L3 cache activity only when L3 cache is enabled
    local stats_dict = ngx.shared.cache_stats
    if data then
        ngx.log(ngx.DEBUG, "L3 cache hit for key: ", cache_key)
        stats_dict:incr("l3_cache_hits", 1, 0)
    else
        ngx.log(ngx.DEBUG, "L3 cache miss for key: ", cache_key)
        stats_dict:incr("l3_cache_misses", 1, 0)
    end
    
    return data
end

-- Helper function to get or create mlcache instance
local function get_cache_instance(cache_type)
    if not cache_instances[cache_type] then
        local shm_name = "rpc_cache"
        if cache_type == "short" then
            shm_name = "rpc_cache_short"
        elseif cache_type == "minimal" then
            shm_name = "rpc_cache_minimal"
        end
        
        local cache, err = mlcache.new("mlcache_" .. cache_type, shm_name, {
            lru_size = 1000,    -- L1 LRU cache size
            ttl = 3600,         -- Default TTL (will be overridden)
            neg_ttl = 30,       -- Negative caching TTL
            resurrect_ttl = 30, -- TTL for stale values during refresh
            ipc_shm = "mlcache_ipc",  -- IPC shared memory for multi-worker sync
            resty_lock_opts = {
                exptime = 30,   -- Lock expiration time
                timeout = 5     -- Lock acquisition timeout
            }
        })
        
        if not cache then
            ngx.log(ngx.ERR, "Failed to create mlcache instance for ", cache_type, ": ", err)
            return nil
        end
        
        cache_instances[cache_type] = cache
    end
    
    return cache_instances[cache_type]
end

-- Helper function to decode JSON once and cache result
local function get_decoded_body(body_data)
    if not body_data then return nil end
    
    -- If already decoded (table), return as is
    if type(body_data) == "table" then
        return body_data
    end
    
    local ok, body_json = pcall(json.decode, body_data)
    if not ok or not body_json or not body_json.method then
        return nil
    end
    
    return body_json
end

-- Unified cache function that handles all cache operations
-- Returns cache information along with decoded request body to avoid duplicate JSON parsing
function _M.check_cache(chain, network, body_data)
    local decoded_body = get_decoded_body(body_data)
    if not decoded_body then
        return {
            cache_type = nil,
            cache_key = nil,
            ttl = nil,
            cached_response = nil,
            decoded_body = nil
        }
    end
    
    local cache_info = cache_rules.get_cache_info(chain, network, decoded_body)
    if not cache_info or cache_info.cache_type == "none" or cache_info.ttl == 0 then
        return {
            cache_type = nil,
            cache_key = nil,
            ttl = nil,
            cached_response = nil,
            decoded_body = decoded_body
        }
    end
    
    local cache_type = cache_info.cache_type
    local ttl = cache_info.ttl
    
    local cache_instance = get_cache_instance(cache_type)
    
    if not cache_instance then
        ngx.log(ngx.ERR, "Failed to get cache instance for type: ", cache_type)
        return {
            cache_type = nil,
            cache_key = nil,
            ttl = nil,
            cached_response = nil,
            decoded_body = decoded_body
        }
    end
    
    local cache_key = chain .. ":" .. network .. ":" .. ngx.md5(body_data)
    
    -- Use L3 callback when getting from cache
    local cached_response, err, hit_level = cache_instance:get(cache_key, { ttl = ttl }, l3_callback, cache_key)
    local stats_dict = ngx.shared.cache_stats
    
    if cached_response and cached_response ~= ngx.null then
        -- Increment cache hit counter based on hit level
        if hit_level == 1 then
            stats_dict:incr("l1_cache_hits_" .. cache_type, 1, 0)
        elseif hit_level == 2 then
            stats_dict:incr("l2_cache_hits_" .. cache_type, 1, 0)
        end
        stats_dict:incr("cache_hits_" .. cache_type, 1, 0)
        stats_dict:incr("cache_hits_total", 1, 0)
    else
        -- Increment cache miss counter
        stats_dict:incr("cache_misses_" .. cache_type, 1, 0)
        stats_dict:incr("cache_misses_total", 1, 0)
        cached_response = nil
    end
    
    -- Increment total requests counter
    stats_dict:incr("total_requests_" .. cache_type, 1, 0)
    stats_dict:incr("total_requests_all", 1, 0)
    
    return {
        cache_type = cache_type,
        cache_key = cache_key,
        ttl = ttl,
        cached_response = cached_response,
        decoded_body = decoded_body,
        cache_instance = cache_instance
    }
end

-- Save function that uses cache_info and also saves to L3 cache
function _M.save_to_cache(cache_info, response_body)
    if not cache_info.cache_type or not cache_info.cache_key or not cache_info.ttl or not cache_info.cache_instance then
        return false
    end
    
    -- Save to L1/L2 cache (mlcache)
    local success, err = cache_info.cache_instance:set(cache_info.cache_key, { ttl = cache_info.ttl }, response_body)
    if not success then
        ngx.log(ngx.ERR, "Failed to cache response to L1/L2 (", cache_info.cache_type, "): ", err)
    end
    
    local l3_success = true  -- Default to true if L3 is disabled
    
    -- Save to L3 cache (KeyDB) only if enabled
    if keydb_l3_cache.enabled() then
        l3_success, l3_err = keydb_l3_cache.set(cache_info.cache_key, response_body, cache_info.ttl)
        if not l3_success then
            ngx.log(ngx.ERR, "Failed to cache response to L3 (", cache_info.cache_type, "): ", l3_err)
        end
    else
        ngx.log(ngx.DEBUG, "L3 cache disabled, skipping save for key: ", cache_info.cache_key)
    end
    
    -- Return true if at least one cache succeeded
    return success or l3_success
end

return _M 