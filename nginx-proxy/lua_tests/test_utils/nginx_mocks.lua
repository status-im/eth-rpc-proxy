local _M = {}

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
        }
    }
end

-- Setup all nginx mocks
function _M.setup_all()
    _M.setup()
end

return _M 