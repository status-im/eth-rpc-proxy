local json = require("cjson")

local _M = {}

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

function _M.is_permanent_method(body_data)
    if not body_data then return false end
    
    local ok, body_json = pcall(json.decode, body_data)
    if not ok or not body_json or not body_json.method then
        return false
    end
    
    return permanent_methods[body_json.method] == true
end

function _M.is_short_method(body_data)
    if not body_data then return false end
    
    local ok, body_json = pcall(json.decode, body_data)
    if not ok or not body_json or not body_json.method then
        return false
    end
    
    return short_methods[body_json.method] == true
end

function _M.is_cacheable_method(body_data)
    return _M.is_permanent_method(body_data) or _M.is_short_method(body_data)
end

function _M.get_cache_key(chain, network, body_data)
    -- Simple hash-like key: chain:network:body_hash
    return chain .. ":" .. network .. ":" .. ngx.crc32_short(body_data)
end

function _M.get_cached_response(cache_key, body_data)
    local shared_dict
    local cache_type
    
    if _M.is_permanent_method(body_data) then
        shared_dict = ngx.shared.rpc_cache
        cache_type = "permanent"
    elseif _M.is_short_method(body_data) then
        shared_dict = ngx.shared.rpc_cache_short
        cache_type = "short"
    else
        return nil
    end
    
    local cached_response = shared_dict:get(cache_key)
    if cached_response then
        ngx.log(ngx.INFO, "Cache hit (", cache_type, ") for key: ", cache_key)
        return cached_response
    end
    ngx.log(ngx.INFO, "Cache miss (", cache_type, ") for key: ", cache_key)
    return nil
end

function _M.save_to_cache(cache_key, response_body, body_data)
    local shared_dict
    local ttl
    local cache_type
    
    if _M.is_permanent_method(body_data) then
        shared_dict = ngx.shared.rpc_cache
        ttl = 86400  -- 24 hours
        cache_type = "permanent"
    elseif _M.is_short_method(body_data) then
        shared_dict = ngx.shared.rpc_cache_short
        ttl = 5      -- 5 seconds
        cache_type = "short"
    else
        ngx.log(ngx.INFO, "Method not cacheable, skipping cache")
        return false
    end
    
    local success, err = shared_dict:set(cache_key, response_body, ttl)
    if success then
        ngx.log(ngx.INFO, "Cached response (", cache_type, ") for key: ", cache_key, " with TTL: ", ttl, " seconds")
    else
        ngx.log(ngx.ERR, "Failed to cache response (", cache_type, "): ", err)
    end
    return success
end

function _M.get_cache_ttl(body_data)
    if _M.is_permanent_method(body_data) then
        return 86400  -- 24 hours
    elseif _M.is_short_method(body_data) then
        return 5      -- 5 seconds
    else
        return nil    -- Not cacheable
    end
end

return _M 