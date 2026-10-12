local _M = {}

-- How to answer a request of a token whose usage count, including this
-- request, is usage: the status and the rate limit headers
function _M.decide(usage, limit, cache_status)
    local allowed = usage <= limit
    return {
        status = allowed and 200 or 401,
        headers = {
            ["X-RateLimit-Limit"] = tostring(limit),
            ["X-RateLimit-Remaining"] = tostring(allowed and limit - usage or 0),
            ["X-Cache-Status"] = cache_status,
        },
    }
end

return _M
