local json = require("cjson")
local http = require("resty.http")

local _M = {}

-- Helper function to make request to cache service via Unix socket
local function make_cache_request(endpoint, method, data)
    local httpc = http.new()
    httpc:set_timeout(5000) -- 5 second timeout

    local body = nil
    local headers = {
        ["Host"] = "cache-service",
        ["Content-Type"] = "application/json",
        ["Connection"] = "keep-alive"
    }
    
    if data then
        body = json.encode(data)
    end

    -- Get cache socket path from environment or use default
    local cache_socket_path = os.getenv("GO_CACHE_SOCKET") or "/tmp/cache.sock"

    -- Make request using the correct method
    local ok, err = httpc:connect("unix:" .. cache_socket_path)
    if not ok then
        ngx.log(ngx.ERR, "[CACHE_UNIX] Failed to connect to socket: ", err)
        return nil, err
    end

    local res, err = httpc:request({
        method = method,
        path = endpoint,
        headers = headers,
        body = body
    })
    
    if not res then
        ngx.log(ngx.ERR, "[CACHE_UNIX] Failed to make request: ", err)
        return nil, err
    end
    
    if res.status ~= 200 then
        ngx.log(ngx.ERR, "[CACHE_UNIX] Cache service returned error status: ", res.status)
        return nil, "HTTP " .. res.status
    end
    
    local response_body = res:read_body()
    if not response_body then
        ngx.log(ngx.ERR, "[CACHE_UNIX] Failed to read response body")
        return nil, "Failed to read cache response"
    end
    
    -- Set keepalive for connection reuse
    httpc:set_keepalive(60000, 10) -- 60s timeout, max 10 connections in pool
    
    local ok, response_data = pcall(json.decode, response_body)
    if not ok then
        ngx.log(ngx.ERR, "[CACHE_UNIX] Failed to decode JSON response: ", response_data)
        return nil, "Invalid JSON response"
    end
    
    return response_data, nil
end

-- Diagnostic function to test cache connectivity
function _M.test_cache_connectivity()
    local test_data = {
        chain = "test",
        network = "test",
        raw_body = '{"jsonrpc":"2.0","method":"test","params":[],"id":999}'
    }
    
    ngx.log(ngx.INFO, "[CACHE_DIAG] Testing Unix socket connectivity...")
    local unix_get_result, unix_get_err = make_cache_request("/cache/get", "POST", test_data)
    if unix_get_result then
        ngx.log(ngx.INFO, "[CACHE_DIAG] Unix socket GET: SUCCESS")

        test_data.data = '{"jsonrpc":"2.0","id":999,"result":"test"}'
        local unix_set_result, unix_set_err = make_cache_request("/cache/set", "POST", test_data)
        if unix_set_result then
            ngx.log(ngx.WARN, "[CACHE_DIAG] Unix socket SET: SUCCESS")
            test_data.data = nil
            local unix_get2_result, unix_get2_err = make_cache_request("/cache/get", "POST", test_data)
            if unix_get2_result and unix_get2_result.found then
                ngx.log(ngx.WARN, "[CACHE_DIAG] Unix socket GET after SET: SUCCESS (HIT)")
            else
                ngx.log(ngx.WARN, "[CACHE_DIAG] Unix socket GET after SET: MISS")
            end
        else
            ngx.log(ngx.ERR, "[CACHE_DIAG] Unix socket SET: FAILED - ", unix_set_err or "unknown error")
        end
    else
        ngx.log(ngx.ERR, "[CACHE_DIAG] Unix socket GET: FAILED - ", unix_get_err or "unknown error")
    end
    
    ngx.log(ngx.WARN, "[CACHE_DIAG] Cache connectivity test completed")
end

return _M
