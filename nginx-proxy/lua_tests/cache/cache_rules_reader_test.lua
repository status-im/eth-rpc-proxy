#!/usr/bin/env lua

-- Use test utilities and nginx mocks
local test_utils = require("test_utils.init")
local nginx_mocks = require("test_utils.nginx_mocks")

-- Setup nginx mocks (direct import)
nginx_mocks.setup_all()

-- Load the module under test
local cache_rules_reader = require("cache_rules_reader")

-- Create test suite
local suite = test_utils.create_test_suite("cache_rules_reader.lua")

-- Run tests
suite:add_test("read_yaml_config with valid file", function(suite)
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
    local temp_file = test_utils.create_temp_file(valid_yaml)
    local config = cache_rules_reader.read_yaml_config(temp_file)
    
    suite:assert_test(config ~= nil, "read_yaml_config returns non-nil for valid file")
    suite:assert_test(config.ttl_defaults ~= nil, "Config contains ttl_defaults")
    suite:assert_test(config.cache_rules ~= nil, "Config contains cache_rules")
    
    test_utils.cleanup_temp_file(temp_file)
end)

suite:add_test("read_yaml_config with non-existent file", function(suite)
    local config_nil = cache_rules_reader.read_yaml_config("/non/existent/file.yaml")
    suite:assert_test(config_nil == nil, "read_yaml_config returns nil for non-existent file")
end)

suite:add_test("read_yaml_config with empty file", function(suite)
    local empty_file = test_utils.create_temp_file("")
    local config_empty = cache_rules_reader.read_yaml_config(empty_file)
    suite:assert_test(config_empty == nil, "read_yaml_config returns nil for empty file")
    test_utils.cleanup_temp_file(empty_file)
end)

suite:add_test("read_yaml_config with invalid YAML", function(suite)
    local invalid_yaml = "invalid_yaml: [\nunclosed bracket"
    local temp_file = test_utils.create_temp_file(invalid_yaml)
    local config_invalid = cache_rules_reader.read_yaml_config(temp_file)
    suite:assert_test(config_invalid == nil, "read_yaml_config returns nil for invalid YAML")
    test_utils.cleanup_temp_file(temp_file)
end)

suite:add_test("validate_config with valid config", function(suite)
    local valid_config = {
        ttl_defaults = {
            default = { permanent = 86400, short = 5, minimal = 0 }
        },
        cache_rules = { eth_blockNumber = "short" }
    }
    local is_valid = cache_rules_reader.validate_config(valid_config)
    suite:assert_test(is_valid == true, "validate_config returns true for valid config")
end)

suite:add_test("validate_config with nil config", function(suite)
    local is_valid_nil = cache_rules_reader.validate_config(nil)
    suite:assert_test(is_valid_nil == false, "validate_config returns false for nil config")
end)

suite:add_test("validate_config with missing ttl_defaults", function(suite)
    local config_missing_ttl = {
        cache_rules = { eth_blockNumber = "short" }
    }
    local is_valid_missing_ttl = cache_rules_reader.validate_config(config_missing_ttl)
    suite:assert_test(is_valid_missing_ttl == false, "validate_config returns false for missing ttl_defaults")
end)

suite:add_test("validate_config with missing cache_rules", function(suite)
    local config_missing_rules = {
        ttl_defaults = {
            default = { permanent = 86400, short = 5, minimal = 0 }
        }
    }
    local is_valid_missing_rules = cache_rules_reader.validate_config(config_missing_rules)
    suite:assert_test(is_valid_missing_rules == false, "validate_config returns false for missing cache_rules")
end)

suite:add_test("validate_config with missing default ttl_defaults", function(suite)
    local config_missing_default = {
        ttl_defaults = {},
        cache_rules = { eth_blockNumber = "short" }
    }
    local is_valid_missing_default = cache_rules_reader.validate_config(config_missing_default)
    suite:assert_test(is_valid_missing_default == false, "validate_config returns false for missing default ttl_defaults")
end)

suite:add_test("validate_config with missing required TTL constants", function(suite)
    local config_missing_ttl = {
        ttl_defaults = {
            default = { permanent = 86400 } -- missing short and minimal
        },
        cache_rules = {}
    }
    local is_valid_missing_ttl = cache_rules_reader.validate_config(config_missing_ttl)
    suite:assert_test(is_valid_missing_ttl == false, "validate_config returns false for missing TTL constants")
end)

local success = suite:run()
os.exit(success and 0 or 1) 