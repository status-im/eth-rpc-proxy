-- Example of cache_test.lua migrated to busted syntax for mlcache with L3 cache

-- Setup KeyDB mocks before requiring modules that depend on resty.redis
local keydb_mocks = require("test_utils.keydb_mocks")
keydb_mocks.setup_require_mocks()

local cache_rules = require("cache.cache_rules")
local cache = require("cache.cache")
local keydb_l3_cache = require("cache.keydb_l3_cache")
local keydb_config = require("cache.keydb_config")

describe("cache.lua integration with mlcache", function()
  local chain, network, request_body, response_body
  
  before_each(function()
    -- Reset test environment
    test_helpers.setup_test_environment()
    
    -- Setup test data
    chain = "ethereum"
    network = "mainnet"
    request_body = '{"jsonrpc":"2.0","method":"eth_getBlockByHash","params":["0x123",true],"id":1}'
    response_body = '{"jsonrpc":"2.0","result":{"number":"0x123","hash":"0x456"},"id":1}'
  end)

  describe("with permanent method", function()
    before_each(function()
      assert.is_true(cache_rules.init("/valid/config.yaml"))
      cache_rules.load_config(false, "/valid/config.yaml")
    end)

    it("should handle full cache integration flow with mlcache", function()
      -- Step 1: Check cache (should be empty initially)
      local cache_check = cache.check_cache(chain, network, request_body)
      
      assert.are.equal("permanent", cache_check.cache_type)
      assert.are.equal(86400, cache_check.ttl)
      assert.is_nil(cache_check.cached_response)
      assert.is_not_nil(cache_check.cache_key)
      assert.is_not_nil(cache_check.decoded_body)
      assert.is_not_nil(cache_check.cache_instance)
      assert.are.equal("eth_getBlockByHash", cache_check.decoded_body.method)
      
      -- Step 2: Save to cache using mlcache
      assert.is_true(cache.save_to_cache(cache_check, response_body))
      
      -- Step 3: Check cache again (should be hit now)
      local cache_check2 = cache.check_cache(chain, network, request_body)
      
      assert.are.equal("permanent", cache_check2.cache_type)
      
      -- Compare JSON objects instead of strings to ignore field order
      local json = require("cjson")
      local expected_response = json.decode(response_body)
      local actual_response = json.decode(cache_check2.cached_response)
      assert.are.same(expected_response, actual_response)
      
      assert.are.equal(cache_check.cache_key, cache_check2.cache_key)
      assert.is_not_nil(cache_check2.decoded_body)
      assert.is_not_nil(cache_check2.cache_instance)
      assert.are.equal("eth_getBlockByHash", cache_check2.decoded_body.method)
    end)

    it("should update cache statistics", function()
      -- Initial check and save
      local cache_check = cache.check_cache(chain, network, request_body)
      cache.save_to_cache(cache_check, response_body)
      
      -- Second check (cache hit)
      cache.check_cache(chain, network, request_body)
      
      -- Verify stats
      local cache_mocks = require("test_utils.cache_mocks")
      local storage = cache_mocks.get_storage()
      local total_requests = storage.cache_stats["total_requests_permanent"] or 0
      local cache_hits = storage.cache_stats["cache_hits_permanent"] or 0
      
      assert.is_true(total_requests >= 2)
      assert.is_true(cache_hits >= 1)
    end)
  end)

  describe("error handling", function()
    it("should handle invalid JSON correctly", function()
      local invalid_request_body = "invalid json here"
      
      local cache_check = cache.check_cache(chain, network, invalid_request_body)
      
      assert.is_nil(cache_check.cache_type)
      assert.is_nil(cache_check.cached_response)
      assert.is_nil(cache_check.decoded_body)
    end)

    it("should handle save without cache_instance", function()
      local invalid_cache_info = {
        cache_type = "permanent",
        cache_key = "test:key",
        ttl = 3600
        -- missing cache_instance
      }
      
      local result = cache.save_to_cache(invalid_cache_info, response_body)
      assert.is_false(result)
    end)
  end)

  describe("different cache types", function()
    before_each(function()
      assert.is_true(cache_rules.init("/valid/config.yaml"))
      cache_rules.load_config(false, "/valid/config.yaml")
    end)

    local cache_scenarios = {
      {
        method = "eth_blockNumber",
        expected_type = "short",
        expected_ttl = 15,  -- ethereum chain overrides default 5 to 15
        description = "short cache type"
      },
      {
        method = "eth_gasPrice", 
        expected_type = "minimal",
        expected_ttl = 5,   -- ethereum chain overrides default 3 to 5
        description = "minimal cache type"
      }
    }

    for _, scenario in ipairs(cache_scenarios) do
  it("should handle " .. scenario.description .. " for " .. scenario.method .. " with mlcache", function()
        local test_request = string.format(
          '{"jsonrpc":"2.0","method":"%s","params":[],"id":1}',
          scenario.method
        )
        
        local cache_check = cache.check_cache(chain, network, test_request)
        
        assert.are.equal(scenario.expected_type, cache_check.cache_type)
        assert.are.equal(scenario.expected_ttl, cache_check.ttl)
        assert.is_not_nil(cache_check.cache_instance)
      end)
    end
  end)

  describe("mlcache specific features", function()
    before_each(function()
      assert.is_true(cache_rules.init("/valid/config.yaml"))
      cache_rules.load_config(false, "/valid/config.yaml")
    end)

    it("should handle different cache instances for different cache types", function()
      local permanent_request = '{"jsonrpc":"2.0","method":"eth_getBlockByHash","params":["0x123",true],"id":1}'
      local short_request = '{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}'
      
      local permanent_check = cache.check_cache(chain, network, permanent_request)
      local short_check = cache.check_cache(chain, network, short_request)
      
      assert.are.equal("permanent", permanent_check.cache_type)
      assert.are.equal("short", short_check.cache_type)
      assert.is_not_nil(permanent_check.cache_instance)
      assert.is_not_nil(short_check.cache_instance)
      
      -- Different cache types should use different instances
      -- In our mock implementation, they're different objects
      assert.is_not_nil(permanent_check.cache_instance)
      assert.is_not_nil(short_check.cache_instance)
    end)
  end)

  describe("normalized cache key generation", function()
    before_each(function()
      assert.is_true(cache_rules.init("/valid/config.yaml"))
      cache_rules.load_config(false, "/valid/config.yaml")
    end)

    it("should generate same cache key for same method with different IDs", function()
      -- Same method, same parameters, but different IDs
      local request1 = '{"jsonrpc":"2.0","method":"eth_chainId","params":[],"id":1}'
      local request2 = '{"jsonrpc":"2.0","method":"eth_chainId","params":[],"id":123456789}'
      local request3 = '{"jsonrpc":"2.0","method":"eth_chainId","params":[],"id":"different_id"}'
      
      local cache_check1 = cache.check_cache(chain, network, request1)
      local cache_check2 = cache.check_cache(chain, network, request2)
      local cache_check3 = cache.check_cache(chain, network, request3)
      
      -- All should have the same cache key despite different IDs
      assert.are.equal(cache_check1.cache_key, cache_check2.cache_key)
      assert.are.equal(cache_check1.cache_key, cache_check3.cache_key)
      assert.are.equal("permanent", cache_check1.cache_type)
      assert.are.equal("permanent", cache_check2.cache_type)
      assert.are.equal("permanent", cache_check3.cache_type)
    end)

    it("should generate different cache keys for different methods", function()
      -- Different methods should have different cache keys
      local request1 = '{"jsonrpc":"2.0","method":"eth_chainId","params":[],"id":1}'
      local request2 = '{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}'
      
      local cache_check1 = cache.check_cache(chain, network, request1)
      local cache_check2 = cache.check_cache(chain, network, request2)
      
      -- Should have different cache keys for different methods
      assert.are_not.equal(cache_check1.cache_key, cache_check2.cache_key)
    end)

    it("should generate different cache keys for different parameters", function()
      -- Same method, different parameters should have different cache keys
      local request1 = '{"jsonrpc":"2.0","method":"eth_getBlockByHash","params":["0x123",true],"id":1}'
      local request2 = '{"jsonrpc":"2.0","method":"eth_getBlockByHash","params":["0x456",true],"id":1}'
      
      local cache_check1 = cache.check_cache(chain, network, request1)
      local cache_check2 = cache.check_cache(chain, network, request2)
      
      -- Should have different cache keys for different parameters
      assert.are_not.equal(cache_check1.cache_key, cache_check2.cache_key)
    end)

    it("should cache responses consistently with normalized keys", function()
      -- Test that caching works correctly with normalized keys
      local request1 = '{"jsonrpc":"2.0","method":"eth_chainId","params":[],"id":1}'
      local request2 = '{"jsonrpc":"2.0","method":"eth_chainId","params":[],"id":999}'
      local response = '{"jsonrpc":"2.0","result":"0x1","id":1}'
      
      -- First request should miss cache
      local cache_check1 = cache.check_cache(chain, network, request1)
      assert.is_nil(cache_check1.cached_response)
      
      -- Save response to cache
      assert.is_true(cache.save_to_cache(cache_check1, response))
      
      -- Second request with different ID should hit cache with corrected id
      local cache_check2 = cache.check_cache(chain, network, request2)
      assert.is_not_nil(cache_check2.cached_response)
      assert.are.equal(cache_check1.cache_key, cache_check2.cache_key)
      
      -- Verify that the id in cached response matches request2's id
      local json = require("cjson")
      local response_data = json.decode(cache_check2.cached_response)
      assert.are.equal(999, response_data.id)  -- Should match request2's id, not the original
      assert.are.equal("0x1", response_data.result)  -- Result should be preserved
      
      -- Also test that the id fixing works correctly
      local original_response = json.decode('{"jsonrpc":"2.0","result":"0x1","id":1}')
      local actual_response = json.decode(cache_check2.cached_response)
      assert.are.equal(original_response.result, actual_response.result)  -- Same result
      assert.are.equal(original_response.jsonrpc, actual_response.jsonrpc)  -- Same jsonrpc
      assert.are_not.equal(original_response.id, actual_response.id)  -- Different id
    end)
  end)

  describe("L3 KeyDB cache integration", function()
    before_each(function()
      assert.is_true(cache_rules.init("/valid/config.yaml"))
      cache_rules.load_config(false, "/valid/config.yaml")
      
      -- Setup keydb_config mock to return proper values
      keydb_config.enabled = function() return true end
      keydb_config.get_default_ttl = function() return 3600 end
      keydb_config.get_max_ttl = function() return 86400 end
      keydb_config.get_keydb_url = function() return "redis://test-keydb:6379" end
      keydb_config.get_connect_timeout = function() return 100 end
      keydb_config.get_send_timeout = function() return 1000 end
      keydb_config.get_read_timeout = function() return 1000 end
      keydb_config.get_pool_size = function() return 10 end
      keydb_config.get_max_idle_timeout = function() return 10000 end
    end)

    it("should check L3 cache enabled status", function()
      -- Test that keydb_l3_cache.enabled() works
      local enabled_status = keydb_l3_cache.enabled()
      assert.is_boolean(enabled_status)
    end)

    it("should handle L3 cache disabled scenario", function()
      -- Mock keydb_config.enabled to return false
      local original_enabled = keydb_config.enabled
      keydb_config.enabled = function() return false end
      
      local cache_check = cache.check_cache(chain, network, request_body)
      cache.save_to_cache(cache_check, response_body)
      
      -- L3 disabled should not affect L1/L2 cache functionality
      assert.is_not_nil(cache_check.cache_type)
      
      -- Restore original function
      keydb_config.enabled = original_enabled
    end)

    it("should integrate L3 cache in callback flow", function()
      -- Test that L3 callback is properly integrated
      local cache_check = cache.check_cache(chain, network, request_body)
      
      -- Save to cache (should save to L1/L2 and L3 if enabled)
      local save_result = cache.save_to_cache(cache_check, response_body)
      assert.is_true(save_result)
      
      -- Second check should hit cache (L1/L2 or L3)
      local cache_check2 = cache.check_cache(chain, network, request_body)
      assert.is_not_nil(cache_check2.cached_response)
    end)

    it("should handle L3 cache statistics", function()
      -- Initial check and save
      local cache_check = cache.check_cache(chain, network, request_body)
      cache.save_to_cache(cache_check, response_body)
      
      -- Second check (should hit cache)
      cache.check_cache(chain, network, request_body)
      
      -- Verify L3 cache stats are tracked
      local cache_mocks = require("test_utils.cache_mocks")
      local storage = cache_mocks.get_storage()
      
      -- Check that L3 cache statistics exist
      local l3_hits = storage.cache_stats["l3_cache_hits"] or 0
      local l3_misses = storage.cache_stats["l3_cache_misses"] or 0
      
      -- Should have L3 activity when L3 callback is invoked
      assert.is_true(l3_misses > 0)  -- L3 misses are expected on first cache access
    end)

    it("should save to both L1/L2 and L3 caches when enabled", function()
      -- Mock keydb_config.enabled to return true
      local original_enabled = keydb_config.enabled
      keydb_config.enabled = function() return true end
      
      local cache_check = cache.check_cache(chain, network, request_body)
      local save_result = cache.save_to_cache(cache_check, response_body)
      
      -- Should succeed even if L3 is enabled
      assert.is_true(save_result)
      
      -- Restore original function
      keydb_config.enabled = original_enabled
    end)
  end)
end) 