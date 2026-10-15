local json = require("cjson")

local _M = {}

-- Share decoding and validation across routing and configured-route projections.
local function decode_chains(text)
    local ok, config = pcall(json.decode, text)
    if not ok then
        return nil, "not JSON: " .. tostring(config)
    end
    -- cjson represents an empty array and an empty object as the same table.
    if type(config) ~= "table" or not text:match("^%s*{") then
        return nil, "not a JSON object"
    end
    local chains = config.chains
    if chains == nil then chains = {} end
    -- A JSON object decodes to a table too, one that ipairs sees as empty
    if type(chains) ~= "table" or (#chains == 0 and next(chains) ~= nil) then
        return nil, "chains is not a list"
    end

    for i, chain in ipairs(chains) do
        if type(chain) ~= "table" or type(chain.name) ~= "string" or chain.name == ""
            or type(chain.network) ~= "string" or chain.network == ""
            or type(chain.providers) ~= "table" then
            return nil, "chain " .. i .. " needs a name, a network and providers"
        end
    end
    return chains
end

-- The providers dict entries a provider config describes: chain:network key
-- to the JSON of its provider list. Malformed configurations are rejected whole.
function _M.parse(text)
    local chains, err = decode_chains(text)
    if not chains then return nil, err end
    local entries = {}
    for _, chain in ipairs(chains) do
        entries[chain.name .. ":" .. chain.network] = json.encode(chain.providers)
    end
    return entries
end

-- A credential-free index of the client's configured route choices. Call this
-- on the complete local configuration, not a health-filtered provider list.
function _M.configured_routes(text)
    local chains, err = decode_chains(text)
    if not chains then return nil, err end
    local routes = {}
    for _, chain in ipairs(chains) do
        local types = { auto = true }
        for _, provider in ipairs(chain.providers) do
            if type(provider) == "table" and type(provider.type) == "string" and provider.type ~= "" then
                types[provider.type] = true
            end
        end
        routes[chain.name .. ":" .. chain.network] = {
            chain = chain.name, network = chain.network, types = types
        }
    end
    return routes
end

-- The stored keys that entries no longer have
function _M.stale(stored_keys, entries)
    local stale = {}
    for _, key in ipairs(stored_keys) do
        if entries[key] == nil then
            stale[#stale + 1] = key
        end
    end
    return stale
end

return _M
