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

function _M.is_permanent_method(body_data)
    if not body_data then return false end
    
    local ok, body_json = pcall(json.decode, body_data)
    if not ok or not body_json or not body_json.method then
        return false
    end
    
    return permanent_methods[body_json.method] == true
end

function _M.get_cache_key(chain, network, body_data)
    -- Simple hash-like key: chain:network:body_hash
    return chain .. ":" .. network .. ":" .. ngx.crc32_short(body_data)
end

function _M.get_cached_response(cache_key)
    local cached_response = ngx.shared.rpc_cache:get(cache_key)
    if cached_response then
        ngx.log(ngx.INFO, "Cache hit for key: ", cache_key)
        return cached_response
    end
    ngx.log(ngx.INFO, "Cache miss for key: ", cache_key)
    return nil
end

function _M.save_to_cache(cache_key, response_body)
    -- Cache for 24 hours (86400 seconds)
    local success, err = ngx.shared.rpc_cache:set(cache_key, response_body, 86400)
    if success then
        ngx.log(ngx.INFO, "Cached response for key: ", cache_key)
    else
        ngx.log(ngx.ERR, "Failed to cache response: ", err)
    end
    return success
end

return _M 