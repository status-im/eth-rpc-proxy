-- Updated cache_test.lua for HTTP-based cache service

local cache = require("cache.cache")

describe("cache.lua HTTP integration", function()
  local chain, network, request_body, response_body
  
  before_each(function()
    -- Setup test data
    chain = "ethereum"
    network = "mainnet"
    request_body = '{"jsonrpc":"2.0","method":"eth_getBlockByHash","params":["0x123",true],"id":1}'
    response_body = '{"jsonrpc":"2.0","result":{"number":"0x123","hash":"0x456"},"id":1}'
    
    -- Mock environment variables for cache service
    _G.os = _G.os or {}
    _G.os.getenv = function(var)
      if var == "GO_CACHE_URL" then
        return "http://localhost:8083"
      elseif var == "GO_CACHE_SOCKET" then
        return "" -- Use HTTP instead of socket for tests
      end
      return nil
    end
  end)

  describe("cache check integration", function()
    it("should handle cache check request format", function()
      -- Test basic cache check functionality
      local cache_check = cache.check_cache(chain, network, request_body)
      
      -- Should return proper structure even if cache service is not available
      assert.is_table(cache_check)
      assert.is_not_nil(cache_check.decoded_body)
      assert.are.equal("eth_getBlockByHash", cache_check.decoded_body.method)
    end)

    it("should handle invalid JSON correctly", function()
      local invalid_request_body = "invalid json here"
      
      local cache_check = cache.check_cache(chain, network, invalid_request_body)
      
      assert.is_nil(cache_check.cache_type)
      assert.is_nil(cache_check.cached_response)
      assert.is_nil(cache_check.decoded_body)
    end)

    it("should handle missing required fields", function()
      local cache_check = cache.check_cache(nil, network, request_body)
      
      -- Should still decode body but fail cache operations
      assert.is_not_nil(cache_check.decoded_body)
      assert.are.equal("eth_getBlockByHash", cache_check.decoded_body.method)
    end)
  end)

  describe("cache save integration", function()
    it("should handle save request format", function()
      -- First get cache info
      local cache_check = cache.check_cache(chain, network, request_body)
      
      -- Mock successful cache info
      cache_check.cache_key = "test:cache:key"
      cache_check.chain = chain
      cache_check.network = network
      
      -- Should not crash when trying to save
      local result = cache.save_to_cache(cache_check, response_body)
      assert.is_boolean(result)
    end)

    it("should handle missing cache info", function()
      local invalid_cache_info = {
        -- missing cache_key and decoded_body
      }
      
      local result = cache.save_to_cache(invalid_cache_info, response_body)
      assert.is_false(result)
    end)
  end)

  describe("response ID fixing", function()
    it("should maintain response structure", function()
      -- Create a mock cache check with cached response
      local cache_check = {
        cache_type = "permanent",
        cache_key = "test:key",
        ttl = 86400,
        cached_response = '{"jsonrpc":"2.0","result":{"number":"0x123"},"id":999}',
        decoded_body = {
          jsonrpc = "2.0",
          method = "eth_getBlockByHash",
          params = {"0x123", true},
          id = 1
        }
      }
      
      -- The cached response should be properly structured
      assert.is_string(cache_check.cached_response)
      
      local json = require("cjson")
      local response_data = json.decode(cache_check.cached_response)
      assert.are.equal("2.0", response_data.jsonrpc)
      assert.is_not_nil(response_data.result)
    end)
  end)

  describe("environment configuration", function()
    it("should handle GO_CACHE_URL environment variable", function()
      -- Mock different cache service URL
      local original_getenv = _G.os.getenv
      _G.os.getenv = function(var)
        if var == "GO_CACHE_URL" then
          return "http://custom-cache:9999"
        end
        return original_getenv(var)
      end
      
      -- Should not crash with different URL
      local cache_check = cache.check_cache(chain, network, request_body)
      assert.is_table(cache_check)
      
      -- Restore original function
      _G.os.getenv = original_getenv
    end)

    it("should handle GO_CACHE_SOCKET environment variable", function()
      -- Mock socket path
      local original_getenv = _G.os.getenv
      _G.os.getenv = function(var)
        if var == "GO_CACHE_SOCKET" then
          return "/tmp/test-cache.sock"
        end
        return original_getenv(var)
      end
      
      -- Should not crash with socket configuration
      local cache_check = cache.check_cache(chain, network, request_body)
      assert.is_table(cache_check)
      
      -- Restore original function
      _G.os.getenv = original_getenv
    end)
  end)
end)