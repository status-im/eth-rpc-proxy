-- route_test.lua
describe("route", function()
    local json = require("cjson")
    local route

    local infura = { type = "infura", url = "https://infura" }
    local alchemy = { type = "alchemy", url = "https://alchemy" }

    setup(function()
        require("spec_helper")
        route = require("utils.route")
    end)

    describe("target", function()
        it("reads the chain, network and provider type from the path", function()
            assert.are.same(
                { chain = "ethereum", network = "mainnet", provider_type = "infura", key = "ethereum:mainnet" },
                route.target("/ethereum/mainnet/infura"))
        end)

        it("leaves the provider type out when the path has none", function()
            assert.are.same(
                { chain = "base", network = "sepolia", key = "base:sepolia" },
                route.target("/base/sepolia"))
        end)

        it("answers a malformed path with 400", function()
            local target, failure = route.target("/ethereum")
            assert.is_nil(target)
            assert.are.equal(400, failure.status)
            assert.are.equal(failure.message, failure.log)
        end)
    end)

    describe("providers", function()
        it("decodes the providers stored for the chain/network", function()
            assert.are.same({ infura, alchemy },
                route.providers(json.encode({ infura, alchemy }), "ethereum:mainnet"))
        end)

        it("answers an unknown chain/network with 404", function()
            local providers, failure = route.providers(nil, "foo:bar")
            assert.is_nil(providers)
            assert.are.equal(404, failure.status)
            assert.are.equal("No providers available for this chain/network", failure.message)
            assert.are.equal("No providers found for foo:bar", failure.log)
        end)

        it("answers stored providers that do not decode with 500", function()
            local providers, failure = route.providers("{ broken", "ethereum:mainnet")
            assert.is_nil(providers)
            assert.are.equal(500, failure.status)
            assert.are.equal("Internal server error: invalid providers configuration", failure.message)
            assert.truthy(failure.log:find("Invalid providers JSON for ethereum:mainnet", 1, true))
        end)
    end)

    describe("candidates", function()
        it("tries every provider when no provider type is asked for", function()
            assert.are.same({ providers = { infura, alchemy }, tried_specific = false },
                route.candidates({ infura, alchemy }, nil))
        end)

        it("tries only the providers of the asked type", function()
            assert.are.same({ providers = { alchemy }, tried_specific = true },
                route.candidates({ infura, alchemy }, "alchemy"))
        end)

        it("answers an empty provider list with 404", function()
            local candidates, failure = route.candidates({}, nil, "ethereum:mainnet")
            assert.is_nil(candidates)
            assert.are.equal(404, failure.status)
            assert.are.equal("No providers available for this chain/network", failure.message)
        end)

        it("answers a provider type nobody has with 404", function()
            local candidates, failure = route.candidates({ infura }, "grove", "ethereum:mainnet")
            assert.is_nil(candidates)
            assert.are.equal(404, failure.status)
            assert.are.equal("No providers available for this provider type", failure.message)
            assert.are.equal("No providers found for provider_type: grove", failure.log)
        end)
    end)

    describe("exhausted", function()
        it("answers 502 when every provider failed", function()
            local failure = route.exhausted(nil, false)
            assert.are.equal(502, failure.status)
            assert.are.equal("All providers failed", failure.message)
            assert.is_nil(failure.log)
        end)

        it("answers 502 when every provider of the asked type failed", function()
            assert.are.equal(502, route.exhausted("infura", true).status)
        end)

        it("answers 404 when the asked type was never tried", function()
            local failure = route.exhausted("infura", false)
            assert.are.equal(404, failure.status)
            assert.are.equal("Provider not found: infura", failure.message)
        end)
    end)
end)
