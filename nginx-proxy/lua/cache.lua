local json = require("cjson")

local _M = {}

-- Default TTLs (in seconds)
local ttl_defaults = {
    default = { permanent = 86400, short = 5, minimal = 0 },
    ["ethereum:mainnet"] = { short = 12, minimal = 5 },
    ["arbitrum:mainnet"] = { short = 1 },
    ["optimism:mainnet"] = { short = 2 },
    ["polygon:mainnet"] = { short = 2 },
}

-- Shared dicts per cache type
local cache_dicts = {
    permanent = ngx.shared.rpc_cache,
    short = ngx.shared.rpc_cache_short,
    minimal = ngx.shared.rpc_cache_minimal
}

-- Permanent methods that can be cached for 24 hours
local permanent_methods = {
    ["eth_getBlockByHash"] = true,
    ["eth_getTransactionByHash"] = true,
    ["eth_getTransactionReceipt"] = true,
    ["eth_getTransactionByBlockHashAndIndex"] = true,
    ["eth_getUncleByBlockHashAndIndex"] = true,
    ["net_version"] = true,
    ["eth_chainId"] = true,
    ["web3_clientVersion"] = true,
    ["net_listening"] = true,
    ["eth_protocolVersion"] = true
}

-- Short-term methods that can be cached for 5 seconds
local short_methods = {
    ["eth_getBlockByNumber"] = true,
    ["eth_getTransactionByBlockNumberAndIndex"] = true,
    ["eth_getUncleByBlockNumberAndIndex"] = true,
    ["eth_getBalance"] = true,
    ["eth_getCode"] = true,
    ["eth_getStorageAt"] = true,
    ["eth_blockNumber"] = true
}

-- Minimal cache methods
local minimal_methods = {
    ["eth_gasPrice"] = true,
    ["eth_maxPriorityFeePerGas"] = true,
    ["eth_feeHistory"] = true
}

-- Helper function to decode JSON once and cache result
local function get_decoded_body(body_data)
    if not body_data then return nil end
    
    -- If already decoded (table), return as is
    if type(body_data) == "table" then
        return body_data
    end
    
    -- If string, decode it
    local ok, body_json = pcall(json.decode, body_data)
    if not ok or not body_json or not body_json.method then
        return nil
    end
    
    return body_json
end

-- Unified cache function that handles all cache operations
function _M.check_cache(chain, network, body_data)
    -- Decode body once
    local decoded_body = get_decoded_body(body_data)
    if not decoded_body then
        return {
            cache_type = nil,
            cache_key = nil,
            ttl = nil,
            cached_response = nil
        }
    end
    
    local method = decoded_body.method
    local cache_type = nil
    
    if permanent_methods[method] then
        cache_type = "permanent"
    elseif short_methods[method] then
        cache_type = "short"
    elseif minimal_methods[method] then
        cache_type = "minimal"
    end
    
    if not cache_type then
        return {
            cache_type = nil,
            cache_key = nil,
            ttl = nil,
            cached_response = nil
        }
    end
    
    -- Generate cache key
    local cache_key = chain .. ":" .. network .. ":" .. ngx.md5(body_data)
    
    -- Fetch TTL config for this chain/network (fallback to default)
    local key = chain .. ":" .. network
    local cfg = ttl_defaults[key] or ttl_defaults.default
    local ttl = cfg[cache_type] or ttl_defaults.default[cache_type]
    
    -- If TTL is 0 or nil, treat as non-cacheable
    if not ttl or ttl == 0 then
        return {
            cache_type = nil,
            cache_key = nil,
            ttl = nil,
            cached_response = nil
        }
    end
    
    local shared_dict = cache_dicts[cache_type]
    
    -- Check for cached response
    local cached_response = shared_dict:get(cache_key)
    local stats_dict = ngx.shared.cache_stats
    
    if cached_response then
        ngx.log(ngx.INFO, "Cache hit (", cache_type, ") for key: ", cache_key)
        -- Increment cache hit counter
        stats_dict:incr("cache_hits_" .. cache_type, 1, 0)
        stats_dict:incr("cache_hits_total", 1, 0)
    else
        ngx.log(ngx.INFO, "Cache miss (", cache_type, ") for key: ", cache_key)
        -- Increment cache miss counter
        stats_dict:incr("cache_misses_" .. cache_type, 1, 0)
        stats_dict:incr("cache_misses_total", 1, 0)
    end
    
    -- Increment total requests counter
    stats_dict:incr("total_requests_" .. cache_type, 1, 0)
    stats_dict:incr("total_requests_all", 1, 0)
    
    return {
        cache_type = cache_type,
        cache_key = cache_key,
        ttl = ttl,
        cached_response = cached_response
    }
end

-- Save function that uses cache_info
function _M.save_to_cache(cache_info, response_body)
    if not cache_info.cache_type or not cache_info.cache_key or not cache_info.ttl then
        return false
    end
    
    local shared_dict = cache_dicts[cache_info.cache_type]
    if not shared_dict then
        ngx.log(ngx.ERR, "Invalid cache_type: ", cache_info.cache_type)
        return false
    end
    
    local success, err = shared_dict:set(cache_info.cache_key, response_body, cache_info.ttl)
    if success then
        ngx.log(ngx.INFO, "Cached response (", cache_info.cache_type, ") for key: ", cache_info.cache_key, " with TTL: ", cache_info.ttl, " seconds")
    else
        ngx.log(ngx.ERR, "Failed to cache response (", cache_info.cache_type, "): ", err)
    end
    return success
end

return _M 
