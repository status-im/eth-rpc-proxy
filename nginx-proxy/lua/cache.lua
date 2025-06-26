local json = require("cjson")

local _M = {}

-- Cache type configurations
local cache_configs = {
    permanent = { dict = ngx.shared.rpc_cache, ttl = 86400 },      -- 24 hours
    short = { dict = ngx.shared.rpc_cache_short, ttl = 5 }         -- 5 seconds
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
    local cache_key = chain .. ":" .. network .. ":" .. ngx.crc32_short(body_data)
    
    -- Get cache configuration
    local config = cache_configs[cache_type]
    local shared_dict = config.dict
    local ttl = config.ttl
    
    -- Check for cached response
    local cached_response = shared_dict:get(cache_key)
    if cached_response then
        ngx.log(ngx.INFO, "Cache hit (", cache_type, ") for key: ", cache_key)
    else
        ngx.log(ngx.INFO, "Cache miss (", cache_type, ") for key: ", cache_key)
    end
    
    return {
        cache_type = cache_type,
        cache_key = cache_key,
        ttl = ttl,
        cached_response = cached_response
    }
end

-- Save function that uses cache_type and mapping
function _M.save_to_cache(cache_key, response_body, cache_type)
    local config = cache_configs[cache_type]
    if not config then
        ngx.log(ngx.ERR, "Invalid cache_type: ", cache_type, ". Expected 'permanent' or 'short'")
        return false
    end
    
    local shared_dict = config.dict
    local ttl = config.ttl
    
    local success, err = shared_dict:set(cache_key, response_body, ttl)
    if success then
        ngx.log(ngx.INFO, "Cached response (", cache_type, ") for key: ", cache_key, " with TTL: ", ttl, " seconds")
    else
        ngx.log(ngx.ERR, "Failed to cache response (", cache_type, "): ", err)
    end
    return success
end

return _M 