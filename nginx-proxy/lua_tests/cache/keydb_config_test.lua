-- keydb_config_test.lua - Tests for keydb_config.lua module
local keydb_mocks = require("test_utils.keydb_mocks")

describe("keydb_config.lua", function()
  local test_env
  local keydb_config

  before_each(function()
    -- Reset test environment
    test_helpers.setup_test_environment()
    
    -- Setup KeyDB test environment with mocks
    test_env = keydb_mocks.setup_keydb_test_environment({
      file_contents = {
        ["/test/config.yaml"] = "valid_yaml"
      }
    })
    
    -- Require the module after mocks are in place
    keydb_config = require("cache.keydb_config")
  end)
  
  after_each(function()
    -- Restore original functions
    if test_env then
      test_env.restore()
    end
    
    -- Clear module from cache
    package.loaded["cache.keydb_config"] = nil
  end)

  describe("initialization", function()
    it("should set default values on init", function()
      keydb_config.init()
      
      -- Should have loaded values from YAML since file exists
      assert.are.equal("redis://test-keydb:6379", keydb_config.get_keydb_url())
      assert.are.equal(200, keydb_config.get_connect_timeout()) -- From YAML
      assert.are.equal(2000, keydb_config.get_send_timeout()) -- From YAML
      assert.are.equal(2000, keydb_config.get_read_timeout()) -- From YAML
      assert.are.equal(20, keydb_config.get_pool_size()) -- From YAML
      assert.are.equal(20000, keydb_config.get_max_idle_timeout()) -- From YAML
      assert.are.equal(7200, keydb_config.get_default_ttl()) -- From YAML
      assert.are.equal(172800, keydb_config.get_max_ttl()) -- From YAML
    end)
    
    it("should use fallback defaults when no config file", function()
      -- Override file mock to simulate missing file
      keydb_mocks.setup_file_mocks({})
      
      -- Clear module cache and reload
      package.loaded["cache.keydb_config"] = nil
      keydb_config = require("cache.keydb_config")
      keydb_config.init()
      
      -- Should use true default values
      assert.are.equal("redis://test-keydb:6379", keydb_config.get_keydb_url())
      assert.are.equal(100, keydb_config.get_connect_timeout()) -- Default
      assert.are.equal(1000, keydb_config.get_send_timeout()) -- Default
      assert.are.equal(1000, keydb_config.get_read_timeout()) -- Default
      assert.are.equal(10, keydb_config.get_pool_size()) -- Default
      assert.are.equal(10000, keydb_config.get_max_idle_timeout()) -- Default
      assert.are.equal(3600, keydb_config.get_default_ttl()) -- Default
      assert.are.equal(86400, keydb_config.get_max_ttl()) -- Default
    end)
  end)

  describe("YAML config loading", function()
    it("should load valid YAML config successfully", function()
      local config_data = keydb_config.read_yaml_config("/test/config.yaml")
      
      assert.is_not_nil(config_data)
      assert.are.equal(200, config_data.connection.connect_timeout)
      assert.are.equal(2000, config_data.connection.send_timeout)
      assert.are.equal(2000, config_data.connection.read_timeout)
      assert.are.equal(20, config_data.keepalive.pool_size)
      assert.are.equal(20000, config_data.keepalive.max_idle_timeout)
      assert.are.equal(7200, config_data.cache.default_ttl)
      assert.are.equal(172800, config_data.cache.max_ttl)
    end)

    it("should handle missing config file", function()
      local config_data = keydb_config.read_yaml_config("/missing/config.yaml")
      
      assert.is_nil(config_data)
    end)

    it("should handle empty config file", function()
      -- Override file mock for this test
      keydb_mocks.setup_file_mocks({
        ["/test/config.yaml"] = ""
      })
      
      local config_data = keydb_config.read_yaml_config("/test/config.yaml")
      
      assert.is_nil(config_data)
    end)

    it("should handle invalid YAML content", function()
      -- Override file mock for this test
      keydb_mocks.setup_file_mocks({
        ["/test/config.yaml"] = "invalid_yaml"
      })
      
      local config_data = keydb_config.read_yaml_config("/test/config.yaml")
      
      assert.is_nil(config_data)
    end)
  end)

  describe("configuration values", function()
    it("should apply configuration from valid YAML", function()
      keydb_config.load_config(false) -- false means not premature
      
      assert.are.equal(200, keydb_config.get_connect_timeout())
      assert.are.equal(2000, keydb_config.get_send_timeout())
      assert.are.equal(2000, keydb_config.get_read_timeout())
      assert.are.equal(20, keydb_config.get_pool_size())
      assert.are.equal(20000, keydb_config.get_max_idle_timeout())
      assert.are.equal(7200, keydb_config.get_default_ttl())
      assert.are.equal(172800, keydb_config.get_max_ttl())
    end)

    it("should use defaults when config loading fails", function()
      -- Override file mock to simulate missing file
      keydb_mocks.setup_file_mocks({})
      
      keydb_config.load_config(false)
      
      assert.are.equal(100, keydb_config.get_connect_timeout())
      assert.are.equal(1000, keydb_config.get_send_timeout())
      assert.are.equal(1000, keydb_config.get_read_timeout())
      assert.are.equal(10, keydb_config.get_pool_size())
      assert.are.equal(10000, keydb_config.get_max_idle_timeout())
      assert.are.equal(3600, keydb_config.get_default_ttl())
      assert.are.equal(86400, keydb_config.get_max_ttl())
    end)

    it("should handle premature timer call", function()
      -- When premature is true, load_config should return early
      keydb_config.load_config(true)
      -- Should not crash or cause errors
    end)
  end)

  describe("environment variable handling", function()
    it("should use default KeyDB URL when env var not set", function()
      -- Override environment mock to return nil for KEYDB_URL
      local original_getenv = os.getenv
      os.getenv = function(key)
        return nil -- No environment variables set
      end
      
      -- Clear module cache and reload
      package.loaded["cache.keydb_config"] = nil
      keydb_config = require("cache.keydb_config")
      keydb_config.init()
      
      -- Resolver converts keydb:6379 to 127.0.0.1:6379
      assert.are.equal("redis://127.0.0.1:6379", keydb_config.get_keydb_url())
      
      -- Restore
      os.getenv = original_getenv
    end)

    it("should strip trailing slash from KeyDB URL", function()
      -- Override environment mock
      local original_getenv = os.getenv
      os.getenv = function(key)
        if key == "KEYDB_URL" then
          return "redis://test-keydb:6379/"
        end
        return nil
      end
      
      -- Clear module cache and reload
      package.loaded["cache.keydb_config"] = nil
      keydb_config = require("cache.keydb_config")
      keydb_config.init()
      
      -- URL should be stripped of trailing slash (resolver doesn't change test-keydb)
      assert.are.equal("redis://test-keydb:6379", keydb_config.get_keydb_url())
      
      -- Restore
      os.getenv = original_getenv
    end)
  end)
end) 