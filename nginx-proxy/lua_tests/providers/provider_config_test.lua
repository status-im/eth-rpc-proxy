-- provider_config_test.lua
describe("provider_config", function()
    local json = require("cjson")
    local provider_config

    local infura = { url = "https://infura", authType = "token-auth", authToken = "t" }
    local alchemy = { url = "https://alchemy", authType = "no-auth" }

    setup(function()
        require("spec_helper")
        provider_config = require("providers.provider_config")
    end)

    local function decoded(entries)
        local out = {}
        for key, stored in pairs(entries) do
            out[key] = json.decode(stored)
        end
        return out
    end

    describe("parse", function()
        it("keys each chain's providers by chain:network", function()
            local entries = provider_config.parse(json.encode({ chains = {
                { name = "ethereum", network = "mainnet", providers = { infura, alchemy } },
                { name = "base", network = "sepolia", providers = { alchemy } },
            } }))

            assert.are.same({
                ["ethereum:mainnet"] = { infura, alchemy },
                ["base:sepolia"] = { alchemy },
            }, decoded(entries))
        end)

        it("reads a config without chains as no providers", function()
            assert.are.same({}, provider_config.parse("{}"))
        end)

        it("rejects text that is not JSON", function()
            local entries, err = provider_config.parse("{ not json")
            assert.is_nil(entries)
            assert.truthy(err)
        end)

        it("rejects JSON that is not an object", function()
            local entries, err = provider_config.parse("[1, 2]")
            assert.is_nil(entries)
            assert.truthy(err)
        end)

        it("rejects chains that are not a list", function()
            local entries, err = provider_config.parse('{"chains": "ethereum"}')
            assert.is_nil(entries)
            assert.truthy(err)
        end)

        it("rejects chains keyed by name instead of listed", function()
            local entries, err = provider_config.parse(
                '{"chains": {"ethereum": {"name": "ethereum", "network": "mainnet", "providers": []}}}')
            assert.is_nil(entries)
            assert.truthy(err)
        end)

        it("rejects a chain that is not an object", function()
            for _, text in ipairs({ '{"chains": [false]}', '{"chains": [null]}', '{"chains": [7]}' }) do
                local entries, err = provider_config.parse(text)
                assert.is_nil(entries, text)
                assert.truthy(err:find("chain 1", 1, true), err)
            end
        end)

        it("rejects the whole config when a chain lacks its name, network or providers", function()
            for _, broken in ipairs({
                { network = "mainnet", providers = {} },
                { name = "ethereum", providers = {} },
                { name = "ethereum", network = "mainnet" },
            }) do
                local entries, err = provider_config.parse(json.encode({ chains = {
                    { name = "base", network = "mainnet", providers = { alchemy } },
                    broken,
                } }))
                assert.is_nil(entries)
                assert.truthy(err:find("chain 2", 1, true), err)
            end
        end)
    end)

    describe("stale", function()
        it("lists the stored keys the new entries no longer have", function()
            local stale = provider_config.stale(
                { "ethereum:mainnet", "linea:mainnet", "base:mainnet" },
                { ["ethereum:mainnet"] = "[]" })
            table.sort(stale)
            assert.are.same({ "base:mainnet", "linea:mainnet" }, stale)
        end)

        it("lists nothing when every stored key stays", function()
            assert.are.same({}, provider_config.stale({ "ethereum:mainnet" }, { ["ethereum:mainnet"] = "[]" }))
        end)
    end)
end)
