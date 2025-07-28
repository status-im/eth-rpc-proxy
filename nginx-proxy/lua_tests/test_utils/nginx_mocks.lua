local _M = {}

-- Simple base64 encoding function for tests
local function encode_base64(input)
    local charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/"
    local result = ""
    local padding = ""
    
    -- For testing purposes, we'll use a simple mock implementation
    if input == "user:pass" then
        return "dXNlcjpwYXNz"
    end
    
    -- Fallback: simple mock implementation 
    local mock_result = ""
    for i = 1, #input do
        local byte = string.byte(input, i)
        mock_result = mock_result .. charset:sub((byte % 64) + 1, (byte % 64) + 1)
    end
    
    -- Add padding to make it look more like base64
    while #mock_result % 4 ~= 0 do
        mock_result = mock_result .. "="
    end
    
    return mock_result
end

-- Nginx mocks
function _M.setup()
    _G.ngx = {
        log = function(level, ...)
            local args = {...}
            local message = table.concat(args, " ")
            local level_str = tostring(level or "LOG")
            print("[" .. level_str .. "] " .. message)
        end,
        timer = {
            at = function(delay, callback, arg)
                callback(false, arg)
                return true, nil
            end
        },
        md5 = function(str)
            return "mock_hash_" .. string.len(str)
        end,
        encode_base64 = encode_base64,
        AF_INET = 2,
        shared = {
            -- These will be populated by cache_mocks.setup_cache_shared_dicts()
        }
    }
end

-- Setup all nginx mocks
function _M.setup_all()
    _M.setup()
end

return _M 