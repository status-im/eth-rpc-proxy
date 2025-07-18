#!/usr/bin/env lua

-- Use common test utilities
local test_utils = require("test_utils.init")

-- Setup nginx mocks
test_utils.setup_nginx_mocks()

-- Mock cache_rules_reader for testing
package.preload["cache.cache_rules_reader"] = function()
    return {
        read_yaml_config = function(file_path)
            if string.match(file_path, "valid") then
                return {
                    ttl_defaults = {
                        default = { permanent = 86400, short = 5, minimal = 3 },
                        ["ethereum:mainnet"] = { short = 15, minimal = 5 }, -- permanent NOT specified, should fall back to default
                        ["arbitrum:mainnet"] = { short = 1 }, -- permanent and minimal NOT specified, should fall back to default
                        ["polygon:mainnet"] = { permanent = 7200, short = 2 }, -- minimal NOT specified, should fall back to default
                        ["bsc:mainnet"] = { permanent = 3600, short = 1, minimal = 0 } -- minimal explicitly set to 0, should NOT fall back to default
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

-- Load the module under test
local cache_rules = require("cache_rules")

-- Create test suite
local suite = test_utils.create_test_suite("cache_rules.lua")

-- Test functions
local function test_init_with_valid_config(suite)
    local init_result = cache_rules.init("/valid/config.yaml")
    suite:assert_test(init_result == true, "init returns true for valid config path")
end

local function test_classify_method_cache_type(suite)
    local cache_type_permanent = cache_rules.classify_method_cache_type("eth_getBlockByHash", {}, "ethereum", "mainnet")
    suite:assert_test(cache_type_permanent == "permanent", "eth_getBlockByHash classified as permanent")

    local cache_type_short = cache_rules.classify_method_cache_type("eth_blockNumber", {}, "ethereum", "mainnet")
    suite:assert_test(cache_type_short == "short", "eth_blockNumber classified as short")

    local cache_type_minimal = cache_rules.classify_method_cache_type("eth_gasPrice", {}, "ethereum", "mainnet")
    suite:assert_test(cache_type_minimal == "minimal", "eth_gasPrice classified as minimal")

    local cache_type_unknown = cache_rules.classify_method_cache_type("unknown_method_name", {}, "ethereum", "mainnet")
    suite:assert_test(cache_type_unknown == nil, "unknown method returns nil")
end

local function test_get_ttl_for_cache_type_default_values(suite)
    local ttl_permanent = cache_rules.get_ttl_for_cache_type("permanent", "ethereum", "testnet")
    suite:assert_test(ttl_permanent == 86400, "permanent TTL returns 86400 for default")

    local ttl_short = cache_rules.get_ttl_for_cache_type("short", "ethereum", "testnet")
    suite:assert_test(ttl_short == 5, "short TTL returns 5 for default")

    local ttl_minimal = cache_rules.get_ttl_for_cache_type("minimal", "ethereum", "testnet")
    suite:assert_test(ttl_minimal == 3, "minimal TTL returns 3 for default")
end

local function test_get_ttl_for_cache_type_network_specific(suite)
    local ttl_short_mainnet = cache_rules.get_ttl_for_cache_type("short", "ethereum", "mainnet")
    suite:assert_test(ttl_short_mainnet == 15, "short TTL returns 15 for ethereum:mainnet")

    local ttl_minimal_mainnet = cache_rules.get_ttl_for_cache_type("minimal", "ethereum", "mainnet")
    suite:assert_test(ttl_minimal_mainnet == 5, "minimal TTL returns 5 for ethereum:mainnet")

    -- Test fallback for missing permanent TTL
    local ttl_permanent_mainnet = cache_rules.get_ttl_for_cache_type("permanent", "ethereum", "mainnet")
    suite:assert_test(ttl_permanent_mainnet == 86400, "permanent TTL falls back to default for ethereum:mainnet")
end

local function test_get_ttl_for_cache_type_zero_values(suite)
    -- Test network with explicit 0 TTL (should not fall back to default)
    local ttl_minimal_bsc = cache_rules.get_ttl_for_cache_type("minimal", "bsc", "mainnet")
    suite:assert_test(ttl_minimal_bsc == 0, "minimal TTL returns 0 for bsc:mainnet (explicitly set)")

    -- Test unknown cache type
    local ttl_unknown = cache_rules.get_ttl_for_cache_type("unknown_type", "ethereum", "mainnet")
    suite:assert_test(ttl_unknown == 0, "unknown cache type returns 0 TTL")
end

local function test_get_cache_info_permanent_methods(suite)
    local decoded_body_permanent = {
        method = "eth_getBlockByHash",
        params = {"0x123", true}
    }
    local cache_info_permanent = cache_rules.get_cache_info("ethereum", "mainnet", decoded_body_permanent)
    suite:assert_test(cache_info_permanent.cache_type == "permanent", "permanent method returns permanent cache type")
    suite:assert_test(cache_info_permanent.ttl == 86400, "permanent method returns correct TTL")
end

local function test_get_cache_info_short_methods(suite)
    local decoded_body_short = {
        method = "eth_blockNumber",
        params = {}
    }
    local cache_info_short = cache_rules.get_cache_info("ethereum", "mainnet", decoded_body_short)
    suite:assert_test(cache_info_short.cache_type == "short", "short method returns short cache type")
    suite:assert_test(cache_info_short.ttl == 15, "short method returns network-specific TTL")
end

local function test_get_cache_info_minimal_methods(suite)
    local decoded_body_minimal = {
        method = "eth_gasPrice",
        params = {}
    }
    -- For ethereum:mainnet, minimal TTL is 5 (non-zero), so it should be cacheable
    local cache_info_minimal = cache_rules.get_cache_info("ethereum", "mainnet", decoded_body_minimal)
    suite:assert_test(cache_info_minimal.cache_type == "minimal", "minimal method with non-zero TTL returns minimal cache type")
    suite:assert_test(cache_info_minimal.ttl == 5, "minimal method returns network-specific TTL")
end

local function test_get_cache_info_fallback_ttl(suite)
    local decoded_body_permanent_fallback = {
        method = "eth_getTransactionReceipt", -- permanent method
        params = {"0x456"}
    }
    -- For ethereum:mainnet, permanent is not specified, so should fall back to default (86400)
    local cache_info_permanent_fallback = cache_rules.get_cache_info("ethereum", "mainnet", decoded_body_permanent_fallback)
    suite:assert_test(cache_info_permanent_fallback.cache_type == "permanent", "permanent method with fallback TTL returns permanent cache type")
    suite:assert_test(cache_info_permanent_fallback.ttl == 86400, "permanent method returns fallback TTL from default")
end

local function test_get_cache_info_minimal_ttl_zero(suite)
    local decoded_body_minimal = {
        method = "eth_gasPrice",
        params = {}
    }
    local cache_info_minimal_zero = cache_rules.get_cache_info("bsc", "mainnet", decoded_body_minimal)
    suite:assert_test(cache_info_minimal_zero.cache_type == "none", "minimal method with explicitly set 0 TTL returns none cache type")
    suite:assert_test(cache_info_minimal_zero.ttl == 0, "minimal method with explicitly set 0 TTL returns 0 TTL")
end

local function test_get_cache_info_unknown_method(suite)
    local decoded_body_unknown = {
        method = "unknown_method_name",
        params = {}
    }
    local cache_info_unknown = cache_rules.get_cache_info("ethereum", "mainnet", decoded_body_unknown)
    suite:assert_test(cache_info_unknown.cache_type == "none", "unknown method returns none cache type")
    suite:assert_test(cache_info_unknown.ttl == 0, "unknown method returns 0 TTL")
end

local function test_get_cache_info_invalid_request_body(suite)
    local cache_info_nil = cache_rules.get_cache_info("ethereum", "mainnet", nil)
    suite:assert_test(cache_info_nil.cache_type == "none", "nil request body returns none cache type")
    suite:assert_test(cache_info_nil.ttl == 0, "nil request body returns 0 TTL")

    local cache_info_no_method = cache_rules.get_cache_info("ethereum", "mainnet", {})
    suite:assert_test(cache_info_no_method.cache_type == "none", "request body without method returns none cache type")
    suite:assert_test(cache_info_no_method.ttl == 0, "request body without method returns 0 TTL")
end

local function test_behavior_after_config_loading_failure(suite)
    -- Reset state by testing init with invalid config (this should trigger load failure in background)
    cache_rules.init("/invalid/config.yaml")
    
    -- Functions should still work with fallback values when config is not loaded
    local fallback_ttl = cache_rules.get_ttl_for_cache_type("permanent", "ethereum", "mainnet")
    -- Note: This will now use fallback values, so it should return 86400
    suite:assert_test(fallback_ttl == 86400, "get_ttl_for_cache_type returns fallback value when config not loaded")
end

-- Run tests with inline functors
suite:add_test("init with valid config", function(suite)
    local init_result = cache_rules.init("/valid/config.yaml")
    suite:assert_test(init_result == true, "init returns true for valid config path")
end)

suite:add_test("classify_method_cache_type", function(suite)
    cache_rules.init("/valid/config.yaml")
    
    suite:assert_test(cache_rules.classify_method_cache_type("eth_getBlockByHash") == "permanent", "eth_getBlockByHash classified as permanent")
    suite:assert_test(cache_rules.classify_method_cache_type("eth_blockNumber") == "short", "eth_blockNumber classified as short")
    suite:assert_test(cache_rules.classify_method_cache_type("eth_gasPrice") == "minimal", "eth_gasPrice classified as minimal")
    suite:assert_test(cache_rules.classify_method_cache_type("unknown_method") == "unknown_type", "unknown method gets unknown_type")
end)

suite:add_test("get_ttl_for_cache_type with default values", function(suite)
    cache_rules.init("/valid/config.yaml")
    
    local permanent_ttl = cache_rules.get_ttl_for_cache_type("permanent", "ethereum", "mainnet")
    local short_ttl = cache_rules.get_ttl_for_cache_type("short", "ethereum", "mainnet") 
    local minimal_ttl = cache_rules.get_ttl_for_cache_type("minimal", "ethereum", "mainnet")
    
    suite:assert_test(permanent_ttl == 86400, "Default permanent TTL is 86400 seconds")
    suite:assert_test(short_ttl == 15, "Network-specific short TTL is 15 seconds")
    suite:assert_test(minimal_ttl == 5, "Network-specific minimal TTL is 5 seconds")
end)

-- Note: More tests can be added in similar inline fashion
-- This demonstrates the new pattern - tests are defined inline as functors

local success = suite:run()
os.exit(success and 0 or 1) 