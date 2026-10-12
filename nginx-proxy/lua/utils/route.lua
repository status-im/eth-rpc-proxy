local json = require("cjson")
local request_utils = require("utils.request_utils")

-- Which providers a request goes to, decided from the request path and the
-- provider lists the shell looked up. A request that cannot be routed gets a
-- failure: the HTTP status and message to answer with, and what to log.
local _M = {}

local function failure(status, message, log)
    return { status = status, message = message, log = log }
end

local NO_CHAIN_PROVIDERS = "No providers available for this chain/network"

-- The chain/network pair and optional provider type a path asks for, with the
-- key its providers are stored under
function _M.target(uri)
    local chain, network, provider_type, err = request_utils.parse_url_path(uri)
    if err then
        return nil, failure(400, err, err)
    end
    return {
        chain = chain,
        network = network,
        provider_type = provider_type,
        key = request_utils.get_chain_network_key(chain, network),
    }
end

-- The providers stored for key, from their stored JSON
function _M.providers(stored, key)
    if not stored then
        return nil, failure(404, NO_CHAIN_PROVIDERS, "No providers found for " .. key)
    end
    local ok, providers = pcall(json.decode, stored)
    if not ok then
        return nil, failure(500, "Internal server error: invalid providers configuration",
            "Invalid providers JSON for " .. key .. ": " .. tostring(providers))
    end
    return providers
end

-- The providers to try in order, and whether the path asked for one type:
-- { providers = ..., tried_specific = ... }
function _M.candidates(providers, provider_type, key)
    if #providers == 0 then
        return nil, failure(404, NO_CHAIN_PROVIDERS, "No providers found for " .. tostring(key))
    end
    local candidates, tried_specific = request_utils.filter_providers(providers, provider_type)
    if #candidates == 0 then
        return nil, failure(404, "No providers available for this provider type",
            "No providers found for provider_type: " .. (provider_type or "none"))
    end
    return { providers = candidates, tried_specific = tried_specific }
end

-- The answer once every candidate has failed
function _M.exhausted(provider_type, tried_specific)
    if provider_type and provider_type ~= "" and not tried_specific then
        return failure(404, "Provider not found: " .. provider_type)
    end
    return failure(502, "All providers failed")
end

return _M
