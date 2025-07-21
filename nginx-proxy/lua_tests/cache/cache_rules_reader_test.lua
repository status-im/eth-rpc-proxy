-- Busted tests for cache_rules_reader.lua  
local cache_rules_reader = require("cache_rules_reader")

describe("cache_rules_reader.lua", function()
  local temp_files = {}
  
  after_each(function()
    -- Cleanup temporary files
    for _, file in ipairs(temp_files) do
      test_helpers.cleanup_temp_file(file)
    end
    temp_files = {}
  end)

  local function create_temp_file_tracked(content)
    local file = test_helpers.create_temp_file(content)
    table.insert(temp_files, file)
    return file
  end

  describe("read_yaml_config", function()
    it("should read valid YAML config", function()
      local valid_yaml = [[
ttl_defaults:
  default:
    permanent: 86400
    short: 5
    minimal: 0
  ethereum:mainnet:
    short: 15
    minimal: 5

cache_rules:
  eth_getBlockByHash: permanent
  eth_blockNumber: short
  eth_gasPrice: minimal
]]
      local temp_file = create_temp_file_tracked(valid_yaml)
      local config = cache_rules_reader.read_yaml_config(temp_file)
      
      assert.is_not_nil(config)
      assert.is_not_nil(config.ttl_defaults)
      assert.is_not_nil(config.cache_rules)
      assert.are.equal(86400, config.ttl_defaults.default.permanent)
      assert.are.equal("permanent", config.cache_rules.eth_getBlockByHash)
    end)

    it("should return nil for non-existent file", function()
      local config = cache_rules_reader.read_yaml_config("/non/existent/file.yaml")
      assert.is_nil(config)
    end)

    it("should return nil for empty file", function()
      local empty_file = create_temp_file_tracked("")
      local config = cache_rules_reader.read_yaml_config(empty_file)
      assert.is_nil(config)
    end)

    it("should return nil for invalid YAML", function()
      local invalid_yaml = "invalid_yaml: [\nunclosed bracket"
      local temp_file = create_temp_file_tracked(invalid_yaml)
      local config = cache_rules_reader.read_yaml_config(temp_file)
      assert.is_nil(config)
    end)

    it("should handle malformed YAML structure", function()
      local malformed_yaml = [[
ttl_defaults: "this should be a table not string"
cache_rules:
  - invalid_list_format
]]
      local temp_file = create_temp_file_tracked(malformed_yaml)
      local config = cache_rules_reader.read_yaml_config(temp_file)
      -- Should parse but validation should fail
      assert.is_not_nil(config)
    end)
  end)

  describe("validate_config", function()
    describe("with valid configurations", function()
      it("should accept complete valid config", function()
        local valid_config = {
          ttl_defaults = {
            default = { permanent = 86400, short = 5, minimal = 0 }
          },
          cache_rules = { eth_blockNumber = "short" }
        }
        local is_valid = cache_rules_reader.validate_config(valid_config)
        assert.is_true(is_valid)
      end)

      it("should accept config with network-specific TTL overrides", function()
        local config_with_overrides = {
          ttl_defaults = {
            default = { permanent = 86400, short = 5, minimal = 0 },
            ["ethereum:mainnet"] = { short = 15, minimal = 5 },
            ["polygon:mainnet"] = { permanent = 7200 }
          },
          cache_rules = { 
            eth_blockNumber = "short",
            eth_gasPrice = "minimal",
            eth_getBlockByHash = "permanent"
          }
        }
        local is_valid = cache_rules_reader.validate_config(config_with_overrides)
        assert.is_true(is_valid)
      end)
    end)

    describe("with invalid configurations", function()
      it("should reject nil config", function()
        local is_valid = cache_rules_reader.validate_config(nil)
        assert.is_false(is_valid)
      end)

      it("should reject config missing ttl_defaults", function()
        local config_missing_ttl = {
          cache_rules = { eth_blockNumber = "short" }
        }
        local is_valid = cache_rules_reader.validate_config(config_missing_ttl)
        assert.is_false(is_valid)
      end)

      it("should reject config missing cache_rules", function()
        local config_missing_rules = {
          ttl_defaults = {
            default = { permanent = 86400, short = 5, minimal = 0 }
          }
        }
        local is_valid = cache_rules_reader.validate_config(config_missing_rules)
        assert.is_false(is_valid)
      end)

      it("should reject config missing default ttl_defaults", function()
        local config_missing_default = {
          ttl_defaults = {},
          cache_rules = { eth_blockNumber = "short" }
        }
        local is_valid = cache_rules_reader.validate_config(config_missing_default)
        assert.is_false(is_valid)
      end)

      local required_ttl_constants = {"permanent", "short", "minimal"}
      
      for _, missing_constant in ipairs(required_ttl_constants) do
        it("should reject config missing required TTL constant: " .. missing_constant, function()
          local incomplete_defaults = { permanent = 86400, short = 5, minimal = 0 }
          incomplete_defaults[missing_constant] = nil
          
          local config_missing_ttl = {
            ttl_defaults = {
              default = incomplete_defaults
            },
            cache_rules = {}
          }
          local is_valid = cache_rules_reader.validate_config(config_missing_ttl)
          assert.is_false(is_valid)
        end)
      end

      it("should reject config with invalid TTL structure", function()
        local config_invalid_structure = {
          ttl_defaults = {
            default = "invalid_string_instead_of_table"
          },
          cache_rules = { eth_blockNumber = "short" }
        }
        local is_valid = cache_rules_reader.validate_config(config_invalid_structure)
        assert.is_false(is_valid)
      end)

             it("should accept config with non-numeric TTL values (no type checking)", function()
         local config_invalid_ttl = {
           ttl_defaults = {
             default = { permanent = "not_a_number", short = 5, minimal = 0 }
           },
           cache_rules = { eth_blockNumber = "short" }
         }
         local is_valid = cache_rules_reader.validate_config(config_invalid_ttl)
         -- Current implementation doesn't validate types, only presence
         assert.is_true(is_valid)
       end)
    end)

    describe("edge cases", function()
      it("should handle empty cache_rules", function()
        local config_empty_rules = {
          ttl_defaults = {
            default = { permanent = 86400, short = 5, minimal = 0 }
          },
          cache_rules = {}
        }
        local is_valid = cache_rules_reader.validate_config(config_empty_rules)
        assert.is_true(is_valid)
      end)

      it("should handle zero TTL values", function()
        local config_zero_ttl = {
          ttl_defaults = {
            default = { permanent = 0, short = 0, minimal = 0 }
          },
          cache_rules = { eth_blockNumber = "short" }
        }
        local is_valid = cache_rules_reader.validate_config(config_zero_ttl)
        assert.is_true(is_valid)
      end)

             it("should accept negative TTL values (no value validation)", function()
         local config_negative_ttl = {
           ttl_defaults = {
             default = { permanent = -1, short = 5, minimal = 0 }
           },
           cache_rules = { eth_blockNumber = "short" }
         }
         local is_valid = cache_rules_reader.validate_config(config_negative_ttl)
         -- Current implementation doesn't validate values, only presence  
         assert.is_true(is_valid)
       end)
    end)
  end)

  describe("integration scenarios", function()
    it("should read and validate a complete configuration file", function()
      local complete_config_yaml = [[
ttl_defaults:
  default:
    permanent: 86400
    short: 5
    minimal: 3
  ethereum:mainnet:
    short: 15
    minimal: 5
  polygon:mainnet:
    permanent: 7200
    short: 2
    minimal: 1

cache_rules:
  eth_getBlockByHash: permanent
  eth_getTransactionReceipt: permanent
  eth_blockNumber: short
  eth_getBalance: short
  eth_gasPrice: minimal
  eth_call: none
]]
      local temp_file = create_temp_file_tracked(complete_config_yaml)
      local config = cache_rules_reader.read_yaml_config(temp_file)
      
      assert.is_not_nil(config)
      assert.is_true(cache_rules_reader.validate_config(config))
      
      -- Verify specific values
      assert.are.equal(86400, config.ttl_defaults.default.permanent)
      assert.are.equal(15, config.ttl_defaults["ethereum:mainnet"].short)
      assert.are.equal("permanent", config.cache_rules.eth_getBlockByHash)
      assert.are.equal("none", config.cache_rules.eth_call)
    end)

    it("should handle read and validate workflow for invalid file", function()
      local invalid_config_yaml = [[
ttl_defaults:
  default:
    permanent: 86400
    # missing short and minimal
cache_rules:
  eth_blockNumber: short
]]
      local temp_file = create_temp_file_tracked(invalid_config_yaml)
      local config = cache_rules_reader.read_yaml_config(temp_file)
      
      assert.is_not_nil(config)
      assert.is_false(cache_rules_reader.validate_config(config))
    end)
  end)
end) 