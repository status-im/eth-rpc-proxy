-- request_utils_test.lua
describe("request_utils", function()
    local request_utils
    
    setup(function()
        -- Load spec helper which sets up all mocks
        require("spec_helper")
        request_utils = require("utils.request_utils")
    end)
    
    before_each(function()
        _G.test_helpers.setup_test_environment()
    end)

    describe("should_retry", function()
        it("should return true when response is nil", function()
            assert.is_true(request_utils.should_retry(nil, nil))
        end)
        
        it("should return true for HTTP retry status codes", function()
            local retry_statuses = {401, 402, 403, 429, 500, 501, 502, 503, 504, 505}
            
            for _, status in ipairs(retry_statuses) do
                local res = {status = status}
                assert.is_true(request_utils.should_retry(res, nil), 
                              "Should retry for status " .. status)
            end
        end)
        
        it("should return false for non-retry HTTP status codes", function()
            local non_retry_statuses = {200, 201, 400, 404, 422}
            
            for _, status in ipairs(non_retry_statuses) do
                local res = {status = status}
                assert.is_false(request_utils.should_retry(res, nil), 
                               "Should not retry for status " .. status)
            end
        end)
        
        it("should return true for JSON-RPC retry error codes", function()
            local retry_codes = {32005, 33000, 33300, 33400}
            
            for _, code in ipairs(retry_codes) do
                local res = {status = 200}
                local decoded = {error = {code = code}}
                assert.is_true(request_utils.should_retry(res, decoded), 
                              "Should retry for JSON-RPC error code " .. code)
            end
        end)
        
        it("should return false for non-retry JSON-RPC error codes", function()
            local res = {status = 200}
            local decoded = {error = {code = -32601}} -- Method not found
            assert.is_false(request_utils.should_retry(res, decoded))
        end)
        
        it("should return false for successful response", function()
            local res = {status = 200}
            local decoded = {result = "success"}
            assert.is_false(request_utils.should_retry(res, decoded))
        end)
    end)

    describe("filter_providers", function()
        local sample_providers
        
        before_each(function()
            sample_providers = {
                {type = "alchemy", url = "https://alchemy.com"},
                {type = "infura", url = "https://infura.com"},
                {type = "alchemy", url = "https://alchemy2.com"},
                {type = "quicknode", url = "https://quicknode.com"}
            }
        end)
        
        it("should return all providers when no provider_type specified", function()
            local filtered, tried_specific = request_utils.filter_providers(sample_providers, nil)
            
            assert.are.equal(4, #filtered)
            assert.is_false(tried_specific)
            assert.are.same(sample_providers, filtered)
        end)
        
        it("should return all providers when empty provider_type specified", function()
            local filtered, tried_specific = request_utils.filter_providers(sample_providers, "")
            
            assert.are.equal(4, #filtered)
            assert.is_false(tried_specific)
            assert.are.same(sample_providers, filtered)
        end)
        
        it("should filter providers by specific type", function()
            local filtered, tried_specific = request_utils.filter_providers(sample_providers, "alchemy")
            
            assert.are.equal(2, #filtered)
            assert.is_true(tried_specific)
            assert.are.equal("alchemy", filtered[1].type)
            assert.are.equal("alchemy", filtered[2].type)
        end)
        
        it("should return empty list for non-existent provider type", function()
            local filtered, tried_specific = request_utils.filter_providers(sample_providers, "nonexistent")
            
            assert.are.equal(0, #filtered)
            assert.is_true(tried_specific)
        end)
        
        it("should handle nil providers input", function()
            local filtered, tried_specific = request_utils.filter_providers(nil, "alchemy")
            
            assert.are.equal(0, #filtered)
            assert.is_false(tried_specific)
        end)
        
        it("should handle non-table providers input", function()
            local filtered, tried_specific = request_utils.filter_providers("not a table", "alchemy")
            
            assert.are.equal(0, #filtered)
            assert.is_false(tried_specific)
        end)
    end)

    describe("parse_url_path", function()
        it("should parse valid chain/network path", function()
            local chain, network, provider_type, err = request_utils.parse_url_path("/ethereum/mainnet")
            
            assert.are.equal("ethereum", chain)
            assert.are.equal("mainnet", network)
            assert.is_nil(provider_type)
            assert.is_nil(err)
        end)
        
        it("should parse valid chain/network/provider_type path", function()
            local chain, network, provider_type, err = request_utils.parse_url_path("/ethereum/mainnet/alchemy")
            
            assert.are.equal("ethereum", chain)
            assert.are.equal("mainnet", network)
            assert.are.equal("alchemy", provider_type)
            assert.is_nil(err)
        end)
        
        it("should parse path with trailing slash", function()
            local chain, network, provider_type, err = request_utils.parse_url_path("/ethereum/mainnet/")
            
            assert.are.equal("ethereum", chain)
            assert.are.equal("mainnet", network)
            assert.is_nil(provider_type)
            assert.is_nil(err)
        end)
        
        it("should return error for invalid path format", function()
            local chain, network, provider_type, err = request_utils.parse_url_path("/ethereum")
            
            assert.is_nil(chain)
            assert.is_nil(network)
            assert.is_nil(provider_type)
            assert.are.equal("Invalid URL format - must be /chain/network or /chain/network/provider_type", err)
        end)
        
        it("should return error for nil URI", function()
            local chain, network, provider_type, err = request_utils.parse_url_path(nil)
            
            assert.is_nil(chain)
            assert.is_nil(network)
            assert.is_nil(provider_type)
            assert.are.equal("Invalid URI", err)
        end)
        
        it("should return error for non-string URI", function()
            local chain, network, provider_type, err = request_utils.parse_url_path(123)
            
            assert.is_nil(chain)
            assert.is_nil(network)
            assert.is_nil(provider_type)
            assert.are.equal("Invalid URI", err)
        end)
    end)

    describe("setup_auth", function()
        it("should handle no authentication", function()
            local provider = {url = "https://example.com"}
            local base_headers = {["Content-Type"] = "application/json"}
            
            local url, headers = request_utils.setup_auth(provider, provider.url, base_headers)
            
            assert.are.equal("https://example.com", url)
            assert.are.equal("application/json", headers["Content-Type"])
            assert.is_nil(headers["Authorization"])
        end)
        
        it("should setup token authentication with trailing slash", function()
            local provider = {
                url = "https://example.com/",
                authType = "token-auth",
                authToken = "abc123"
            }
            local base_headers = {["Content-Type"] = "application/json"}
            
            local url, headers = request_utils.setup_auth(provider, provider.url, base_headers)
            
            assert.are.equal("https://example.com/abc123", url)
            assert.are.equal("application/json", headers["Content-Type"])
        end)
        
        it("should setup token authentication without trailing slash", function()
            local provider = {
                url = "https://example.com",
                authType = "token-auth",
                authToken = "abc123"
            }
            local base_headers = {["Content-Type"] = "application/json"}
            
            local url, headers = request_utils.setup_auth(provider, provider.url, base_headers)
            
            assert.are.equal("https://example.com/abc123", url)
            assert.are.equal("application/json", headers["Content-Type"])
        end)
        
        it("should setup basic authentication", function()
            local provider = {
                url = "https://example.com",
                authType = "basic-auth",
                authLogin = "user",
                authPassword = "pass"
            }
            local base_headers = {["Content-Type"] = "application/json"}
            
            local url, headers = request_utils.setup_auth(provider, provider.url, base_headers)
            
            assert.are.equal("https://example.com", url)
            assert.are.equal("application/json", headers["Content-Type"])
            -- user:pass in base64 is dXNlcjpwYXNz
            assert.are.equal("Basic dXNlcjpwYXNz", headers["Authorization"])
        end)
        
        it("should handle nil provider", function()
            local url, headers = request_utils.setup_auth(nil, "https://example.com", {})
            
            assert.are.equal("https://example.com", url)
            assert.are.same({}, headers)
        end)
        
        it("should handle missing auth fields", function()
            local provider = {
                url = "https://example.com",
                authType = "token-auth"
                -- missing authToken
            }
            local base_headers = {["Content-Type"] = "application/json"}
            
            local url, headers = request_utils.setup_auth(provider, provider.url, base_headers)
            
            assert.are.equal("https://example.com", url)
            assert.are.equal("application/json", headers["Content-Type"])
            assert.is_nil(headers["Authorization"])
        end)
    end)

    describe("filter_response_headers", function()
        it("should filter out blocklisted headers", function()
            local headers = {
                ["Content-Type"] = "application/json",
                ["Connection"] = "keep-alive",
                ["Transfer-Encoding"] = "chunked",
                ["Access-Control-Allow-Origin"] = "*",
                ["Custom-Header"] = "value"
            }
            
            local filtered = request_utils.filter_response_headers(headers)
            
            assert.are.equal("application/json", filtered["Content-Type"])
            assert.are.equal("value", filtered["Custom-Header"])
            assert.is_nil(filtered["Connection"])
            assert.is_nil(filtered["Transfer-Encoding"])
            assert.is_nil(filtered["Access-Control-Allow-Origin"])
        end)
        
        it("should handle case-insensitive header names", function()
            local headers = {
                ["content-type"] = "application/json",
                ["CONNECTION"] = "keep-alive",
                ["Content-Length"] = "100"
            }
            
            local filtered = request_utils.filter_response_headers(headers)
            
            assert.are.equal("application/json", filtered["content-type"])
            assert.are.equal("100", filtered["Content-Length"])
            assert.is_nil(filtered["CONNECTION"])
        end)
        
        it("should handle nil headers", function()
            local filtered = request_utils.filter_response_headers(nil)
            assert.are.same({}, filtered)
        end)
        
        it("should handle non-table headers", function()
            local filtered = request_utils.filter_response_headers("not a table")
            assert.are.same({}, filtered)
        end)
    end)

    describe("get_chain_network_key", function()
        it("should create chain:network key", function()
            local key = request_utils.get_chain_network_key("ethereum", "mainnet")
            assert.are.equal("ethereum:mainnet", key)
        end)
        
        it("should return nil for missing chain", function()
            local key = request_utils.get_chain_network_key(nil, "mainnet")
            assert.is_nil(key)
        end)
        
        it("should return nil for missing network", function()
            local key = request_utils.get_chain_network_key("ethereum", nil)
            assert.is_nil(key)
        end)
        
        it("should handle empty strings", function()
            local key = request_utils.get_chain_network_key("", "mainnet")
            assert.are.equal(":mainnet", key)
        end)
    end)

    describe("constants", function()
        it("should have correct retry status codes", function()
            assert.is_true(request_utils.RETRY_STATUS[500])
            assert.is_true(request_utils.RETRY_STATUS[502])
            assert.is_true(request_utils.RETRY_STATUS[429])
            assert.is_nil(request_utils.RETRY_STATUS[200])
            assert.is_nil(request_utils.RETRY_STATUS[404])
        end)
        
        it("should have correct retry EVM error codes", function()
            assert.is_true(request_utils.RETRY_EVM[32005])
            assert.is_true(request_utils.RETRY_EVM[33000])
            assert.is_true(request_utils.RETRY_EVM[33300])
            assert.is_true(request_utils.RETRY_EVM[33400])
            assert.is_nil(request_utils.RETRY_EVM[32601])
        end)
        
        it("should have correct header blocklist", function()
            assert.is_true(request_utils.HEADER_BLOCKLIST["connection"])
            assert.is_true(request_utils.HEADER_BLOCKLIST["access-control-allow-origin"])
            assert.is_true(request_utils.HEADER_BLOCKLIST["transfer-encoding"])
            assert.is_nil(request_utils.HEADER_BLOCKLIST["content-type"])
            assert.is_nil(request_utils.HEADER_BLOCKLIST["content-length"])
        end)
    end)
end) 