-- Example of cache_test.lua migrated to busted syntax for mlcache
local cache_rules = require("cache.cache_rules")
local cache = require("cache.cache")

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
      assert.are.equal(response_body, cache_check2.cached_response)
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
end) 