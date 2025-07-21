local _M = {}

local storage = {
    rpc_cache = {},
    rpc_cache_short = {},
    rpc_cache_minimal = {},
    cache_stats = {}
}

-- Mock shared dict interface
local function create_shared_dict_mock(storage_table)
    return {
        get = function(_, key)
            return storage_table[key]
        end,
        set = function(_, key, value, ttl)
            storage_table[key] = value
            return true, nil
        end,
        incr = function(_, key, value, init)
            storage_table[key] = (storage_table[key] or init or 0) + value
            return storage_table[key], nil
        end,
        flush_all = function(_)
            for k in pairs(storage_table) do
                storage_table[k] = nil
            end
        end
    }
end

function _M.setup_cache_shared_dicts()
    if not _G.ngx then
        error("nginx mocks must be setup first")
    end
    
    _G.ngx.shared = _G.ngx.shared or {}
    _G.ngx.shared.rpc_cache = create_shared_dict_mock(storage.rpc_cache)
    _G.ngx.shared.rpc_cache_short = create_shared_dict_mock(storage.rpc_cache_short)
    _G.ngx.shared.rpc_cache_minimal = create_shared_dict_mock(storage.rpc_cache_minimal)
    _G.ngx.shared.cache_stats = create_shared_dict_mock(storage.cache_stats)
end

function _M.clear_cache_storage()
    for dict_name, dict_storage in pairs(storage) do
        for k in pairs(dict_storage) do
            dict_storage[k] = nil
        end
    end
end

function _M.get_storage()
    return storage
end

-- Mock cache_rules_reader with valid config
function _M.setup_cache_rules_reader_mock()
    package.preload["cache.cache_rules_reader"] = function()
        return {
            read_yaml_config = function(file_path)
                if string.match(file_path, "valid") then
                    return {
                        ttl_defaults = {
                            default = { permanent = 86400, short = 5, minimal = 3 },
                            ["ethereum:mainnet"] = { short = 15, minimal = 5 },
                            ["polygon:mainnet"] = { permanent = 7200, short = 2 },
                            ["arbitrum:mainnet"] = { short = 1 },
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
                return nil
            end,
            validate_config = function(config)
                return config ~= nil and config.ttl_defaults ~= nil and config.cache_rules ~= nil
            end
        }
    end
end

function _M.setup_all()
    _M.setup_cache_shared_dicts()
    _M.setup_cache_rules_reader_mock()
end

function _M.reset_all()
    _M.clear_cache_storage()
    -- Don't call setup_all() here - it will be called by spec_helper
end

return _M 