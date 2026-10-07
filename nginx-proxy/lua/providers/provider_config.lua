local json = require("cjson")

local _M = {}

-- The providers dict entries a provider config describes: chain:network key
-- to the JSON of its provider list. A config with any malformed chain is
-- rejected whole, so a reload never applies half of it.
function _M.parse(text)
    local ok, config = pcall(json.decode, text)
    if not ok then
        return nil, "not JSON: " .. tostring(config)
    end
    if type(config) ~= "table" or config[1] ~= nil then
        return nil, "not a JSON object"
    end
    local chains = config.chains or {}
    -- A JSON object decodes to a table too, one that ipairs sees as empty
    if type(chains) ~= "table" or (#chains == 0 and next(chains) ~= nil) then
        return nil, "chains is not a list"
    end

    local entries = {}
    for i, chain in ipairs(chains) do
        if type(chain) ~= "table" or type(chain.name) ~= "string" or type(chain.network) ~= "string"
            or type(chain.providers) ~= "table" then
            return nil, "chain " .. i .. " needs a name, a network and providers"
        end
        entries[chain.name .. ":" .. chain.network] = json.encode(chain.providers)
    end
    return entries
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
