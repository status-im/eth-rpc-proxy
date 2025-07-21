#!/usr/bin/env lua

local test_utils = require("test_utils.init")
local cache_mocks = require("test_utils.cache_mocks")

test_utils.setup_nginx_mocks()
cache_mocks.setup_all()

local cache_rules = require("cache.cache_rules")
local cache = require("cache.cache")

local suite = test_utils.create_test_suite("cache.lua integration")

suite:add_test("cache integration flow with permanent method", function(suite)
    local init_result = cache_rules.init("/valid/config.yaml")
    suite:assert_test(init_result == true, "cache_rules.init succeeds")
    cache_rules.load_config(false, "/valid/config.yaml")
    
    local chain = "ethereum"
    local network = "mainnet"
    local request_body = '{"jsonrpc":"2.0","method":"eth_getBlockByHash","params":["0x123",true],"id":1}'
    local response_body = '{"jsonrpc":"2.0","result":{"number":"0x123","hash":"0x456"},"id":1}'
    
    -- Step 1: Check cache (should be empty initially)
    local cache_check = cache.check_cache(chain, network, request_body)
    suite:assert_test(cache_check.cache_type == "permanent", "cache type is permanent")
    suite:assert_test(cache_check.ttl == 86400, "TTL is 86400 for permanent cache type")
    suite:assert_test(cache_check.cached_response == nil, "no cached response initially")
    suite:assert_test(cache_check.cache_key ~= nil, "cache key is generated")
    
    -- Step 2: Save to cache
    local save_result = cache.save_to_cache(cache_check, response_body)
    suite:assert_test(save_result == true, "save_to_cache succeeds")
    
    -- Step 3: Check cache again (should be hit now)
    local cache_check2 = cache.check_cache(chain, network, request_body)
    suite:assert_test(cache_check2.cache_type == "permanent", "cache type still permanent")
    suite:assert_test(cache_check2.cached_response == response_body, "cached response matches saved response")
    suite:assert_test(cache_check2.cache_key == cache_check.cache_key, "cache key is consistent")
    
    -- Step 4: Verify cache stats were updated
    local storage = cache_mocks.get_storage()
    local total_requests = storage.cache_stats["total_requests_permanent"] or 0
    local cache_hits = storage.cache_stats["cache_hits_permanent"] or 0
    suite:assert_test(total_requests >= 2, "total requests counter incremented")
    suite:assert_test(cache_hits >= 1, "cache hits counter incremented")
    
    print("✓ Full cache integration flow tested successfully")
end)

local success = suite:run()
os.exit(success and 0 or 1) 