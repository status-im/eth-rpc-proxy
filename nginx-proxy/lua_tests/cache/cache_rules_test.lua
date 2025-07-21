-- Busted tests for cache_rules.lua
local cache_rules = require("cache_rules")

describe("cache_rules.lua", function()
  
  before_each(function()
    -- Reset test environment
    test_helpers.setup_test_environment()
    
    -- Clear existing modules to force reload
    package.loaded["cache_rules"] = nil
    package.loaded["cache.cache_rules_reader"] = nil
    
    -- Override cache_mocks to provide our test config
    local cache_mocks = require("test_utils.cache_mocks")
    cache_mocks.setup_cache_rules_reader_mock = function()
      package.preload["cache.cache_rules_reader"] = function()
        return {
          read_yaml_config = function(file_path)
            if string.match(file_path, "valid") then
              return {
                ttl_defaults = {
                  default = { permanent = 86400, short = 5, minimal = 3 },
                  ["ethereum:mainnet"] = { short = 15, minimal = 5 },
                  ["arbitrum:mainnet"] = { short = 1 },
                  ["polygon:mainnet"] = { permanent = 7200, short = 2 },
                  ["bsc:mainnet"] = { permanent = 3600, short = 1, minimal = 0 }
                },
                cache_rules = {
                  eth_getBlockByHash = "permanent",
                  eth_getTransactionReceipt = "permanent",
                  eth_blockNumber = "short",
                  eth_getBalance = "short",
                  eth_gasPrice = "minimal",
                  unknown_method = "unknown_type"
                }
              }
            elseif string.match(file_path, "invalid") then
              return nil
            end
          end,
          validate_config = function(config)
            return config ~= nil and config.ttl_defaults ~= nil and config.cache_rules ~= nil
          end
        }
      end
    end
    
    -- Setup all mocks again with our override
    cache_mocks.setup_all()
  end)

  describe("init", function()
    it("should return true for valid config path", function()
      local init_result = cache_rules.init("/valid/config.yaml")
      assert.is_true(init_result)
    end)

    it("should handle invalid config gracefully", function()
      local init_result = cache_rules.init("/invalid/config.yaml")
      -- Should not crash, might return false or use fallback values
      assert.is_not_nil(init_result)
    end)
  end)

  describe("classify_method_cache_type", function()
    before_each(function()
      cache_rules.init("/valid/config.yaml")
    end)

    it("should classify permanent methods correctly", function()
      local cache_type = cache_rules.classify_method_cache_type("eth_getBlockByHash")
      assert.are.equal("permanent", cache_type)
      
      cache_type = cache_rules.classify_method_cache_type("eth_getTransactionReceipt")
      assert.are.equal("permanent", cache_type)
    end)

    it("should classify short methods correctly", function()
      local cache_type = cache_rules.classify_method_cache_type("eth_blockNumber")
      assert.are.equal("short", cache_type)
      
      cache_type = cache_rules.classify_method_cache_type("eth_getBalance")
      assert.are.equal("short", cache_type)
    end)

    it("should classify minimal methods correctly", function()
      local cache_type = cache_rules.classify_method_cache_type("eth_gasPrice")
      assert.are.equal("minimal", cache_type)
    end)

    it("should handle unknown methods", function()
      local cache_type = cache_rules.classify_method_cache_type("unknown_method")
      assert.are.equal("unknown_type", cache_type)
      
      cache_type = cache_rules.classify_method_cache_type("completely_unknown_method")
      assert.is_nil(cache_type)
    end)
  end)

  describe("get_ttl_for_cache_type", function()
    before_each(function()
      cache_rules.init("/valid/config.yaml")
    end)

    describe("with default values", function()
      it("should return correct default TTL values", function()
        local ttl_permanent = cache_rules.get_ttl_for_cache_type("permanent", "ethereum", "testnet")
        assert.are.equal(86400, ttl_permanent)

        local ttl_short = cache_rules.get_ttl_for_cache_type("short", "ethereum", "testnet")
        assert.are.equal(5, ttl_short)

        local ttl_minimal = cache_rules.get_ttl_for_cache_type("minimal", "ethereum", "testnet")
        assert.are.equal(3, ttl_minimal)
      end)
    end)

    describe("with network-specific overrides", function()
      it("should return network-specific TTL for ethereum:mainnet", function()
        local ttl_short = cache_rules.get_ttl_for_cache_type("short", "ethereum", "mainnet")
        assert.are.equal(15, ttl_short)

        local ttl_minimal = cache_rules.get_ttl_for_cache_type("minimal", "ethereum", "mainnet")
        assert.are.equal(5, ttl_minimal)
      end)

      it("should fall back to default for missing network-specific values", function()
        -- For ethereum:mainnet, permanent is not specified, should fall back to default
        local ttl_permanent = cache_rules.get_ttl_for_cache_type("permanent", "ethereum", "mainnet")
        assert.are.equal(86400, ttl_permanent)
      end)

      it("should handle partial network overrides correctly", function()
        -- arbitrum:mainnet only has short=1, others should fall back
        local ttl_short = cache_rules.get_ttl_for_cache_type("short", "arbitrum", "mainnet")
        assert.are.equal(1, ttl_short)
        
        local ttl_permanent = cache_rules.get_ttl_for_cache_type("permanent", "arbitrum", "mainnet")
        assert.are.equal(86400, ttl_permanent) -- fallback to default
        
        local ttl_minimal = cache_rules.get_ttl_for_cache_type("minimal", "arbitrum", "mainnet")
        assert.are.equal(3, ttl_minimal) -- fallback to default
      end)

      it("should handle polygon:mainnet overrides correctly", function()
        local ttl_permanent = cache_rules.get_ttl_for_cache_type("permanent", "polygon", "mainnet")
        assert.are.equal(7200, ttl_permanent)
        
        local ttl_short = cache_rules.get_ttl_for_cache_type("short", "polygon", "mainnet")
        assert.are.equal(2, ttl_short)
        
        -- minimal not specified, should fall back
        local ttl_minimal = cache_rules.get_ttl_for_cache_type("minimal", "polygon", "mainnet")
        assert.are.equal(3, ttl_minimal)
      end)
    end)

    describe("with zero and special values", function()
             it("should handle explicitly set zero TTL", function()
         -- bsc:mainnet has minimal=0 explicitly set
         local ttl_minimal = cache_rules.get_ttl_for_cache_type("minimal", "bsc", "mainnet")
         assert.are.equal(0, ttl_minimal)
       end)

      it("should return 0 for unknown cache types", function()
        local ttl_unknown = cache_rules.get_ttl_for_cache_type("unknown_type", "ethereum", "mainnet")
        assert.are.equal(0, ttl_unknown)
      end)
    end)
  end)

  describe("get_cache_info", function()
    before_each(function()
      cache_rules.init("/valid/config.yaml")
    end)

    describe("for permanent methods", function()
      it("should return correct cache info for eth_getBlockByHash", function()
        local decoded_body = {
          method = "eth_getBlockByHash",
          params = {"0x123", true}
        }
        local cache_info = cache_rules.get_cache_info("ethereum", "mainnet", decoded_body)
        
        assert.are.equal("permanent", cache_info.cache_type)
        assert.are.equal(86400, cache_info.ttl) -- fallback to default for ethereum:mainnet
      end)

      it("should return correct cache info for eth_getTransactionReceipt", function()
        local decoded_body = {
          method = "eth_getTransactionReceipt",
          params = {"0x456"}
        }
        local cache_info = cache_rules.get_cache_info("ethereum", "mainnet", decoded_body)
        
        assert.are.equal("permanent", cache_info.cache_type)
        assert.are.equal(86400, cache_info.ttl) -- fallback to default
      end)
    end)

    describe("for short methods", function()
      it("should return correct cache info for eth_blockNumber", function()
        local decoded_body = {
          method = "eth_blockNumber",
          params = {}
        }
        local cache_info = cache_rules.get_cache_info("ethereum", "mainnet", decoded_body)
        
        assert.are.equal("short", cache_info.cache_type)
        assert.are.equal(15, cache_info.ttl) -- network-specific override
      end)

      it("should return correct cache info for eth_getBalance", function()
        local decoded_body = {
          method = "eth_getBalance",
          params = {"0x123", "latest"}
        }
        local cache_info = cache_rules.get_cache_info("ethereum", "mainnet", decoded_body)
        
        assert.are.equal("short", cache_info.cache_type)
        assert.are.equal(15, cache_info.ttl)
      end)
    end)

    describe("for minimal methods", function()
      it("should return correct cache info when TTL is non-zero", function()
        local decoded_body = {
          method = "eth_gasPrice",
          params = {}
        }
        local cache_info = cache_rules.get_cache_info("ethereum", "mainnet", decoded_body)
        
        assert.are.equal("minimal", cache_info.cache_type)
        assert.are.equal(5, cache_info.ttl) -- network-specific override
      end)

      it("should return none cache type when TTL is zero", function()
        local decoded_body = {
          method = "eth_gasPrice",
          params = {}
        }
        local cache_info = cache_rules.get_cache_info("bsc", "mainnet", decoded_body)
        
        assert.are.equal("none", cache_info.cache_type)
        assert.are.equal(0, cache_info.ttl) -- explicitly set to 0
      end)
    end)

    describe("for unknown methods", function()
      it("should return none cache type for completely unknown methods", function()
        local decoded_body = {
          method = "unknown_method_name",
          params = {}
        }
        local cache_info = cache_rules.get_cache_info("ethereum", "mainnet", decoded_body)
        
        assert.are.equal("none", cache_info.cache_type)
        assert.are.equal(0, cache_info.ttl)
      end)

      it("should handle methods with unknown_type classification", function()
        local decoded_body = {
          method = "unknown_method", -- This maps to "unknown_type" in config
          params = {}
        }
        local cache_info = cache_rules.get_cache_info("ethereum", "mainnet", decoded_body)
        
        assert.are.equal("none", cache_info.cache_type)
        assert.are.equal(0, cache_info.ttl)
      end)
    end)

    describe("error handling", function()
      it("should handle nil request body", function()
        local cache_info = cache_rules.get_cache_info("ethereum", "mainnet", nil)
        
        assert.are.equal("none", cache_info.cache_type)
        assert.are.equal(0, cache_info.ttl)
      end)

      it("should handle request body without method", function()
        local cache_info = cache_rules.get_cache_info("ethereum", "mainnet", {})
        
        assert.are.equal("none", cache_info.cache_type)
        assert.are.equal(0, cache_info.ttl)
      end)

      it("should handle request body with empty method", function()
        local decoded_body = {
          method = "",
          params = {}
        }
        local cache_info = cache_rules.get_cache_info("ethereum", "mainnet", decoded_body)
        
        assert.are.equal("none", cache_info.cache_type)
        assert.are.equal(0, cache_info.ttl)
      end)
    end)
  end)

  describe("fallback behavior", function()
    it("should work with fallback values when config loading fails", function()
      -- Reset state by testing init with invalid config
      cache_rules.init("/invalid/config.yaml")
      
      -- Functions should still work with fallback values
      local fallback_ttl = cache_rules.get_ttl_for_cache_type("permanent", "ethereum", "mainnet")
      assert.are.equal(86400, fallback_ttl) -- Should use fallback default
    end)

    it("should handle missing network configurations gracefully", function()
      cache_rules.init("/valid/config.yaml")
      
      -- Test with completely unknown network
      local ttl = cache_rules.get_ttl_for_cache_type("short", "unknown_chain", "unknown_network")
      assert.are.equal(5, ttl) -- Should fall back to default
    end)
  end)

  describe("integration scenarios", function()
    before_each(function()
      cache_rules.init("/valid/config.yaml")
    end)

    it("should handle complex ethereum:mainnet scenario", function()
      -- Test permanent method (fallback to default)
      local permanent_body = { method = "eth_getBlockByHash", params = {"0x123", true} }
      local permanent_info = cache_rules.get_cache_info("ethereum", "mainnet", permanent_body)
      assert.are.equal("permanent", permanent_info.cache_type)
      assert.are.equal(86400, permanent_info.ttl)
      
      -- Test short method (network override)
      local short_body = { method = "eth_blockNumber", params = {} }
      local short_info = cache_rules.get_cache_info("ethereum", "mainnet", short_body)
      assert.are.equal("short", short_info.cache_type)
      assert.are.equal(15, short_info.ttl)
      
      -- Test minimal method (network override)
      local minimal_body = { method = "eth_gasPrice", params = {} }
      local minimal_info = cache_rules.get_cache_info("ethereum", "mainnet", minimal_body)
      assert.are.equal("minimal", minimal_info.cache_type)
      assert.are.equal(5, minimal_info.ttl)
    end)

    it("should handle bsc:mainnet zero TTL scenario", function()
      local minimal_body = { method = "eth_gasPrice", params = {} }
      local cache_info = cache_rules.get_cache_info("bsc", "mainnet", minimal_body)
      
      -- Should return none because TTL is explicitly 0
      assert.are.equal("none", cache_info.cache_type)
      assert.are.equal(0, cache_info.ttl)
    end)

    it("should handle polygon:mainnet partial overrides", function()
      -- Test permanent (has override)
      local permanent_body = { method = "eth_getBlockByHash", params = {"0x123", true} }
      local permanent_info = cache_rules.get_cache_info("polygon", "mainnet", permanent_body)
      assert.are.equal("permanent", permanent_info.cache_type)
      assert.are.equal(7200, permanent_info.ttl)
      
      -- Test short (has override)
      local short_body = { method = "eth_blockNumber", params = {} }
      local short_info = cache_rules.get_cache_info("polygon", "mainnet", short_body)
      assert.are.equal("short", short_info.cache_type)
      assert.are.equal(2, short_info.ttl)
      
      -- Test minimal (fallback to default)
      local minimal_body = { method = "eth_gasPrice", params = {} }
      local minimal_info = cache_rules.get_cache_info("polygon", "mainnet", minimal_body)
      assert.are.equal("minimal", minimal_info.cache_type)
      assert.are.equal(3, minimal_info.ttl) -- fallback to default
    end)
  end)
end) 