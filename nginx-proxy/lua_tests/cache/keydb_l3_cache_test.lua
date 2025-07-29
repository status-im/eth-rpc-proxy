-- keydb_l3_cache_test.lua - Tests for keydb_l3_cache.lua module

-- Fix for Lua 5.4 compatibility
local unpack = table.unpack or unpack

describe("keydb_l3_cache.lua", function()
  local keydb_l3_cache
  local redis_mock_instance
  local original_require

  before_each(function()
    -- Reset test environment
    test_helpers.setup_test_environment()
    
    -- Create simple redis mock
    local redis_state = {
      connect_called = false,
      connect_host = nil,
      connect_port = nil,
      connect_result = {true, nil},
      set_timeouts_called = false,
      set_timeouts_args = {},
      setex_called = false,
      setex_args = {},
      setex_result = {true, nil},
      get_called = false,
      get_args = {},
      get_result = {nil, nil},
      set_keepalive_called = false,
      set_keepalive_args = {},
      set_keepalive_result = {true, nil}
    }
    
    redis_mock_instance = {
      state = redis_state,
      new = function()
        return {
          set_timeouts = function(self, connect_timeout, send_timeout, read_timeout)
            redis_state.set_timeouts_called = true
            redis_state.set_timeouts_args = {connect_timeout, send_timeout, read_timeout}
          end,
          connect = function(self, host, port)
            redis_state.connect_called = true
            redis_state.connect_host = host
            redis_state.connect_port = port
            return unpack(redis_state.connect_result)
          end,
          setex = function(self, key, ttl, value)
            redis_state.setex_called = true
            redis_state.setex_args = {key, ttl, value}
            return unpack(redis_state.setex_result)
          end,
          get = function(self, key)
            redis_state.get_called = true
            redis_state.get_args = {key}
            return unpack(redis_state.get_result)
          end,
          set_keepalive = function(self, max_idle_timeout, pool_size)
            redis_state.set_keepalive_called = true
            redis_state.set_keepalive_args = {max_idle_timeout, pool_size}
            return unpack(redis_state.set_keepalive_result)
          end
        }
      end
    }
    
    local config_mock = {
      get_keydb_url = function() return "redis://test-keydb:6379" end,
      get_connect_timeout = function() return 100 end,
      get_send_timeout = function() return 1000 end,
      get_read_timeout = function() return 1000 end,
      get_pool_size = function() return 10 end,
      get_max_idle_timeout = function() return 10000 end,
      get_default_ttl = function() return 3600 end,
      get_max_ttl = function() return 86400 end
    }
    
    -- Override require
    original_require = require
    _G.require = function(module_name)
      if module_name == "resty.redis" then
        return redis_mock_instance
      elseif module_name == "cache.keydb_config" then
        return config_mock
      else
        return original_require(module_name)
      end
    end
    
    -- Require the module after mocks are in place
    keydb_l3_cache = require("cache.keydb_l3_cache")
  end)
  
  after_each(function()
    -- Restore original require function
    _G.require = original_require
    
    -- Clear module from cache
    package.loaded["cache.keydb_l3_cache"] = nil
    
    -- Reset mock states
    redis_mock_instance = nil
  end)

  describe("set operation", function()
    it("should successfully set a value with default TTL", function()
      redis_mock_instance.state.setex_result = {true, nil}
      
      local success, err = keydb_l3_cache.set("test_key", "test_value")
      
      assert.is_true(success)
      assert.is_nil(err)
      assert.is_true(redis_mock_instance.state.connect_called)
      assert.are.equal("test-keydb", redis_mock_instance.state.connect_host)
      assert.are.equal(6379, redis_mock_instance.state.connect_port)
      assert.is_true(redis_mock_instance.state.setex_called)
      assert.are.equal("test_key", redis_mock_instance.state.setex_args[1])
      assert.are.equal(3600, redis_mock_instance.state.setex_args[2]) -- default TTL
      assert.are.equal("test_value", redis_mock_instance.state.setex_args[3])
      assert.is_true(redis_mock_instance.state.set_keepalive_called)
    end)

    it("should set a value with custom TTL", function()
      redis_mock_instance.state.setex_result = {true, nil}
      
      local success, err = keydb_l3_cache.set("test_key", "test_value", 7200)
      
      assert.is_true(success)
      assert.is_nil(err)
      assert.are.equal(7200, redis_mock_instance.state.setex_args[2])
    end)

    it("should respect max TTL limit", function()
      redis_mock_instance.state.setex_result = {true, nil}
      
      local success, err = keydb_l3_cache.set("test_key", "test_value", 100000) -- Larger than max TTL
      
      assert.is_true(success)
      assert.is_nil(err)
      assert.are.equal(86400, redis_mock_instance.state.setex_args[2]) -- Should be capped to max TTL
    end)

    it("should handle missing key", function()
      local success, err = keydb_l3_cache.set(nil, "test_value")
      
      assert.is_false(success)
      assert.are.equal("Key and value are required", err)
      assert.is_false(redis_mock_instance.state.connect_called)
    end)

    it("should handle missing value", function()
      local success, err = keydb_l3_cache.set("test_key", nil)
      
      assert.is_false(success)
      assert.are.equal("Key and value are required", err)
      assert.is_false(redis_mock_instance.state.connect_called)
    end)

    it("should handle non-string value", function()
      local success, err = keydb_l3_cache.set("test_key", {value = "test"})
      
      assert.is_false(success)
      assert.are.equal("Invalid data format", err)
      assert.is_true(redis_mock_instance.state.connect_called)
      assert.is_true(redis_mock_instance.state.set_keepalive_called)
    end)

    it("should handle connection failure", function()
      redis_mock_instance.state.connect_result = {false, "Connection failed"}
      
      local success, err = keydb_l3_cache.set("test_key", "test_value")
      
      assert.is_false(success)
      assert.are.equal("Connection failed", err)
      assert.is_true(redis_mock_instance.state.connect_called)
      assert.is_false(redis_mock_instance.state.setex_called)
    end)

    it("should handle setex failure", function()
      redis_mock_instance.state.setex_result = {false, "Set operation failed"}
      
      local success, err = keydb_l3_cache.set("test_key", "test_value")
      
      assert.is_false(success)
      assert.are.equal("Set operation failed", err)
      assert.is_true(redis_mock_instance.state.setex_called)
    end)
  end)

  describe("get operation", function()
    it("should successfully get a cached value", function()
      redis_mock_instance.state.get_result = {"cached_value", nil}
      
      local value, err = keydb_l3_cache.get("test_key")
      
      assert.are.equal("cached_value", value)
      assert.is_nil(err)
      assert.is_true(redis_mock_instance.state.connect_called)
      assert.is_true(redis_mock_instance.state.get_called)
      assert.are.equal("test_key", redis_mock_instance.state.get_args[1])
      assert.is_true(redis_mock_instance.state.set_keepalive_called)
    end)

    it("should handle cache miss", function()
      -- Ensure ngx.null is available
      if not ngx.null then
        ngx.null = {}
      end
      redis_mock_instance.state.get_result = {ngx.null, nil}
      
      local value, err = keydb_l3_cache.get("test_key")
      
      assert.is_nil(value)
      assert.are.equal("cache miss", err)
      assert.is_true(redis_mock_instance.state.get_called)
    end)

    it("should handle missing key", function()
      local value, err = keydb_l3_cache.get(nil)
      
      assert.is_nil(value)
      assert.are.equal("Key is required", err)
      assert.is_false(redis_mock_instance.state.connect_called)
    end)

    it("should handle connection failure", function()
      redis_mock_instance.state.connect_result = {false, "Connection failed"}
      
      local value, err = keydb_l3_cache.get("test_key")
      
      assert.is_nil(value)
      assert.are.equal("Connection failed", err)
      assert.is_true(redis_mock_instance.state.connect_called)
      assert.is_false(redis_mock_instance.state.get_called)
    end)

    it("should handle get operation failure", function()
      redis_mock_instance.state.get_result = {nil, "Get operation failed"}
      
      local value, err = keydb_l3_cache.get("test_key")
      
      assert.is_nil(value)
      assert.are.equal("Get operation failed", err)
      assert.is_true(redis_mock_instance.state.get_called)
    end)
  end)

  describe("connection management", function()
    it("should set correct timeouts", function()
      redis_mock_instance.state.setex_result = {true, nil}
      keydb_l3_cache.set("test_key", "test_value")
      
      assert.is_true(redis_mock_instance.state.set_timeouts_called)
      assert.are.equal(100, redis_mock_instance.state.set_timeouts_args[1]) -- connect_timeout
      assert.are.equal(1000, redis_mock_instance.state.set_timeouts_args[2]) -- send_timeout
      assert.are.equal(1000, redis_mock_instance.state.set_timeouts_args[3]) -- read_timeout
    end)

    it("should use correct keepalive parameters", function()
      redis_mock_instance.state.setex_result = {true, nil}
      keydb_l3_cache.set("test_key", "test_value")
      
      assert.is_true(redis_mock_instance.state.set_keepalive_called)
      assert.are.equal(10000, redis_mock_instance.state.set_keepalive_args[1]) -- max_idle_timeout
      assert.are.equal(10, redis_mock_instance.state.set_keepalive_args[2]) -- pool_size
    end)

    it("should handle keepalive failure gracefully", function()
      redis_mock_instance.state.setex_result = {true, nil}
      redis_mock_instance.state.set_keepalive_result = {false, "Keepalive failed"}
      
      local success, err = keydb_l3_cache.set("test_key", "test_value")
      
      -- Should still succeed even if keepalive fails
      assert.is_true(success)
      assert.is_nil(err)
    end)
  end)

  describe("URL parsing", function()
    it("should parse Redis URL with port", function()
      -- Override keydb_config mock for this test
      local custom_config = {
        get_keydb_url = function() return "redis://custom-host:1234" end,
        get_connect_timeout = function() return 100 end,
        get_send_timeout = function() return 1000 end,
        get_read_timeout = function() return 1000 end,
        get_pool_size = function() return 10 end,
        get_max_idle_timeout = function() return 10000 end,
        get_default_ttl = function() return 3600 end,
        get_max_ttl = function() return 86400 end
      }
      
      -- Override require for this test
      _G.require = function(module_name)
        if module_name == "resty.redis" then
          return redis_mock_instance
        elseif module_name == "cache.keydb_config" then
          return custom_config
        else
          return original_require(module_name)
        end
      end
      
      -- Clear module cache and reload with new config
      package.loaded["cache.keydb_l3_cache"] = nil
      local test_keydb_l3_cache = require("cache.keydb_l3_cache")
      
      redis_mock_instance.state.setex_result = {true, nil}
      test_keydb_l3_cache.set("test_key", "test_value")
      
      assert.are.equal("custom-host", redis_mock_instance.state.connect_host)
      assert.are.equal(1234, redis_mock_instance.state.connect_port)
    end)

    it("should use default port when not specified", function()
      -- Override keydb_config mock for this test
      local custom_config = {
        get_keydb_url = function() return "redis://custom-host" end,
        get_connect_timeout = function() return 100 end,
        get_send_timeout = function() return 1000 end,
        get_read_timeout = function() return 1000 end,
        get_pool_size = function() return 10 end,
        get_max_idle_timeout = function() return 10000 end,
        get_default_ttl = function() return 3600 end,
        get_max_ttl = function() return 86400 end
      }
      
      -- Override require for this test
      _G.require = function(module_name)
        if module_name == "resty.redis" then
          return redis_mock_instance
        elseif module_name == "cache.keydb_config" then
          return custom_config
        else
          return original_require(module_name)
        end
      end
      
      -- Clear module cache and reload with new config
      package.loaded["cache.keydb_l3_cache"] = nil
      local test_keydb_l3_cache = require("cache.keydb_l3_cache")
      
      redis_mock_instance.state.setex_result = {true, nil}
      test_keydb_l3_cache.set("test_key", "test_value")
      
      assert.are.equal("custom-host", redis_mock_instance.state.connect_host)
      assert.are.equal(6379, redis_mock_instance.state.connect_port)
    end)

    it("should handle invalid URL format", function()
      -- Override keydb_config mock for this test
      local custom_config = {
        get_keydb_url = function() return "invalid-url" end,
        get_connect_timeout = function() return 100 end,
        get_send_timeout = function() return 1000 end,
        get_read_timeout = function() return 1000 end,
        get_pool_size = function() return 10 end,
        get_max_idle_timeout = function() return 10000 end,
        get_default_ttl = function() return 3600 end,
        get_max_ttl = function() return 86400 end
      }
      
      -- Override require for this test
      _G.require = function(module_name)
        if module_name == "resty.redis" then
          return redis_mock_instance
        elseif module_name == "cache.keydb_config" then
          return custom_config
        else
          return original_require(module_name)
        end
      end
      
      -- Clear module cache and reload with new config
      package.loaded["cache.keydb_l3_cache"] = nil
      local test_keydb_l3_cache = require("cache.keydb_l3_cache")
      
      local success, err = test_keydb_l3_cache.set("test_key", "test_value")
      
      assert.is_false(success)
      assert.are.equal("Invalid URL format", err)
    end)
  end)
end) 