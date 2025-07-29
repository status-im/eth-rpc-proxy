local json = require("cjson")
local cache_rules = require("cache.cache_rules")
local mlcache = require("resty.mlcache")

local _M = {}

-- Initialize mlcache instances for different cache types
local cache_instances = {}

-- Function to reset cache instances (for testing)
function _M.reset_cache_instances()
    cache_instances = {}
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
    
    local cached_response, err = cache_instance:get(cache_key, nil, nil)
    local stats_dict = ngx.shared.cache_stats
    
    if cached_response and cached_response ~= null then
        -- Increment cache hit counter
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

-- Save function that uses cache_info
function _M.save_to_cache(cache_info, response_body)
    if not cache_info.cache_type or not cache_info.cache_key or not cache_info.ttl or not cache_info.cache_instance then
        return false
    end
    
    local success, err = cache_info.cache_instance:set(cache_info.cache_key, cache_info.ttl, response_body)
    if not success then
        ngx.log(ngx.ERR, "Failed to cache response (", cache_info.cache_type, "): ", err)
        return false
    end
    
    return true
end

return _M 