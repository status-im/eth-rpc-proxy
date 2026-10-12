-- provider_loader_test.lua
describe("provider_loader", function()
    local json = require("cjson")
    local provider_loader
    local saved_ngx, config_file, logged

    -- Shared dict that, after every write, lets a concurrent reader look up
    -- the given keys, the way another worker may serve a request mid-reload
    local function observed_dict(watched_keys)
        local data = {}
        local dict = { data = data, missing_seen = {}, writes = {}, no_room_for = {} }
        local function observe()
            for _, key in ipairs(watched_keys) do
                if data[key] == nil then
                    dict.missing_seen[#dict.missing_seen + 1] = key
                end
            end
        end
        function dict:get(key) return data[key] end
        function dict:safe_set(key, value)
            self.writes[#self.writes + 1] = "set " .. key
            if self.no_room_for[key] then return nil, "no memory" end
            data[key] = value; observe(); return true
        end
        function dict:delete(key)
            self.writes[#self.writes + 1] = "delete " .. key
            data[key] = nil; observe()
        end
        function dict:flush_all()
            for key in pairs(data) do data[key] = nil end
            observe()
        end
        function dict:get_keys()
            local keys = {}
            for key in pairs(data) do keys[#keys + 1] = key end
            return keys
        end
        return dict
    end

    local function write_config(chains)
        local file = io.open(config_file, "w")
        file:write(json.encode({ chains = chains }))
        file:close()
    end

    local function chain(name, network, urls)
        local providers = {}
        for i, url in ipairs(urls) do
            providers[i] = { url = url, authType = "no-auth" }
        end
        return { name = name, network = network, providers = providers }
    end

    local function urls_of(dict, key)
        local stored = dict.data[key]
        if not stored then return nil end
        local urls = {}
        for i, provider in ipairs(json.decode(stored)) do
            urls[i] = provider.url
        end
        return urls
    end

    setup(function()
        require("spec_helper")
        provider_loader = require("providers.provider_loader")
        config_file = os.tmpname()
    end)

    teardown(function()
        os.remove(config_file)
    end)

    before_each(function()
        saved_ngx = _G.ngx
        logged = {}
        _G.ngx = setmetatable({ shared = {}, INFO = "info", ERR = "err", log = function(level, ...)
            logged[#logged + 1] = level .. ": " .. table.concat({ ... })
        end }, { __index = saved_ngx })
    end)

    after_each(function()
        _G.ngx = saved_ngx
    end)

    local function reload()
        provider_loader.reload_providers(false, nil, config_file)
    end

    it("never leaves a chain that stays configured without providers", function()
        local dict = observed_dict({ "ethereum:mainnet", "base:mainnet" })
        ngx.shared.providers = dict
        write_config({
            chain("ethereum", "mainnet", { "https://a" }),
            chain("base", "mainnet", { "https://b" }),
        })
        reload()
        dict.missing_seen = {}

        write_config({
            chain("ethereum", "mainnet", { "https://a2" }),
            chain("base", "mainnet", { "https://b2" }),
        })
        reload()

        assert.are.same({}, dict.missing_seen)
        assert.are.same({ "https://a2" }, urls_of(dict, "ethereum:mainnet"))
        assert.are.same({ "https://b2" }, urls_of(dict, "base:mainnet"))
    end)

    it("removes a chain that is no longer configured", function()
        local dict = observed_dict({})
        ngx.shared.providers = dict
        write_config({
            chain("ethereum", "mainnet", { "https://a" }),
            chain("linea", "mainnet", { "https://l" }),
        })
        reload()

        write_config({ chain("ethereum", "mainnet", { "https://a" }) })
        reload()

        assert.is_nil(dict.data["linea:mainnet"])
        assert.are.same({ "https://a" }, urls_of(dict, "ethereum:mainnet"))
    end)

    it("makes room by removing dropped chains before it writes", function()
        local dict = observed_dict({})
        ngx.shared.providers = dict
        write_config({ chain("linea", "mainnet", { "https://l" }) })
        reload()
        dict.writes = {}

        write_config({ chain("ethereum", "mainnet", { "https://a" }) })
        reload()

        assert.are.same({ "delete linea:mainnet", "set ethereum:mainnet" }, dict.writes)
    end)

    it("reports a chain it has no room for and stores the others", function()
        local dict = observed_dict({})
        ngx.shared.providers = dict
        dict.no_room_for["base:mainnet"] = true
        write_config({
            chain("ethereum", "mainnet", { "https://a" }),
            chain("base", "mainnet", { "https://b" }),
        })

        reload()

        assert.are.same({ "https://a" }, urls_of(dict, "ethereum:mainnet"))
        local report = table.concat(logged, "\n")
        assert.truthy(report:find("err: Failed to store providers for base:mainnet: no memory", 1, true), report)
        assert.is_nil(report:find("Providers reloaded and stored", 1, true))
    end)

    it("keeps the current providers when the new config does not parse", function()
        local dict = observed_dict({})
        ngx.shared.providers = dict
        write_config({ chain("ethereum", "mainnet", { "https://a" }) })
        reload()

        local file = io.open(config_file, "w")
        file:write("{ not json")
        file:close()
        reload()

        assert.are.same({ "https://a" }, urls_of(dict, "ethereum:mainnet"))
    end)
end)
