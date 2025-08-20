local json = require("cjson")
local http = require("resty.http")

local _M = {}

-- Cache service configuration
local cache_socket_path = os.getenv("GO_CACHE_SOCKET") or "/tmp/cache.sock"

-- Helper function to make request to cache service via Unix socket
local function make_cache_request(endpoint, method, data)
    local httpc = http.new()
    httpc:set_timeout(5000) -- 5 second timeout
    
    local body = nil
    local headers = {
        ["Content-Type"] = "application/json"
    }
    
    if data then
        body = json.encode(data)
    end
    
    -- Connect to Unix socket
    local ok, connect_err = httpc:connect("unix:" .. cache_socket_path)
    if not ok then
        ngx.log(ngx.ERR, "Failed to connect to cache service via Unix socket: ", connect_err)
        return nil, connect_err
    end
    
    -- Make request via Unix socket
    local res, err = httpc:request({
        path = endpoint,
        method = method,
        body = body,
        headers = headers
    })
    
    if not res then
        ngx.log(ngx.ERR, "Failed to connect to cache service: ", err)
        return nil, err
    end
    
    if res.status ~= 200 then
        ngx.log(ngx.ERR, "Cache service returned status: ", res.status, " body: ", res.body or "")
        return nil, "HTTP " .. res.status
    end
    
    local ok, response_data = pcall(json.decode, res.body)
    if not ok then
        ngx.log(ngx.ERR, "Failed to decode cache service response: ", response_data)
        return nil, "Invalid JSON response"
    end
    
    httpc:close()
    return response_data, nil
end

-- Helper function to validate raw body
local function validate_body(body_data)
    if not body_data or body_data == "" then 
        return false
    end
    
    -- Basic validation - should be JSON string
    if type(body_data) ~= "string" then
        return false
    end
    
    return true
end



-- Check cache for a request
function _M.check_cache(chain, network, body_data)
    ngx.log(ngx.INFO, "[CACHE_DEBUG] check_cache called - chain: ", chain, ", network: ", network, ", body_length: ", body_data and #body_data or "nil")
    
    if not validate_body(body_data) then
        ngx.log(ngx.WARN, "[CACHE_DEBUG] Invalid body data")
        return {
            cache_type = nil,
            cache_key = nil,
            ttl = nil,
            cached_response = nil,
            raw_body = nil
        }
    end
    
    -- Prepare request for cache service with raw body
    local cache_request = {
        chain = chain,
        network = network,
        raw_body = body_data
    }
    
    -- Make GET request to cache service
    local response, err = make_cache_request("/cache/get", "POST", cache_request)
    if not response then
        ngx.log(ngx.ERR, "[CACHE_DEBUG] Failed to get from cache service: ", err)
        return {
            cache_type = nil,
            cache_key = nil,
            ttl = nil,
            cached_response = nil,
            raw_body = body_data
        }
    end
    
    if not response.success then
        ngx.log(ngx.ERR, "[CACHE_DEBUG] Cache service returned error: ", response.error or "unknown")
        return {
            cache_type = nil,
            cache_key = nil,
            ttl = nil,
            cached_response = nil,
            raw_body = body_data
        }
    end
    
    local cached_response = nil
    local cache_status = response.cache_status or "MISS"
    
    if response.found and response.data then
        cached_response = response.data
    end
    
    return {
        cache_type = response.cache_type,
        cache_key = response.key,
        ttl = response.ttl,
        cached_response = cached_response,
        raw_body = body_data,
        fresh = response.fresh,
        cache_status = cache_status
    }
end

-- Save response to cache
function _M.save_to_cache(cache_info, response_body)
    ngx.log(ngx.INFO, "[CACHE_DEBUG] save_to_cache called - cache_key: ", cache_info.cache_key or "nil")
    
    if not cache_info.cache_key or not cache_info.raw_body then
        ngx.log(ngx.WARN, "[CACHE_DEBUG] Missing cache info parameters, cannot save to cache")
        return false
    end
    
    -- Prepare request for cache service
    local cache_request = {
        chain = cache_info.chain or "",
        network = cache_info.network or "",
        raw_body = cache_info.raw_body,
        data = response_body
    }
    
    -- Add TTL if provided
    if cache_info.ttl then
        cache_request.ttl = cache_info.ttl
    end
    
    -- Make SET request to cache service
    local response, err = make_cache_request("/cache/set", "POST", cache_request)
    if not response then
        ngx.log(ngx.ERR, "[CACHE_DEBUG] Failed to save to cache service: ", err)
        return false
    end
    
    if not response.success then
        ngx.log(ngx.ERR, "[CACHE_DEBUG] Cache service save failed: ", response.error or "unknown")
        return false
    end
    
    ngx.log(ngx.INFO, "[CACHE_DEBUG] Successfully saved to cache - key: ", response.key or "unknown")
    return true
end

-- Function to reset cache instances (for testing compatibility)
function _M.reset_cache_instances()
    -- No-op for HTTP-based cache
    ngx.log(ngx.INFO, "[CACHE_DEBUG] reset_cache_instances called (no-op for HTTP cache)")
end

return _M