local json = require("cjson")
local http = require("resty.http")
local cache_diagnostics = require("utils.cache_diagnostics")

local _M = {}

-- Cache service configuration
local cache_socket_path = os.getenv("GO_CACHE_SOCKET") or "/tmp/cache.sock"

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
    if not validate_body(body_data) then
        ngx.log(ngx.WARN, "[CACHE_DEBUG] Invalid body data")
        return {
            cache_type = nil,
            cache_key = nil,
            ttl = nil,
            cached_response = nil,
            raw_body = nil,
            chain = chain,
            network = network
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
        ngx.log(ngx.ERR, "[CACHE_GET] Failed to get from cache service: ", err)
        return {
            cache_type = nil,
            cache_key = nil,
            ttl = nil,
            cached_response = nil,
            raw_body = body_data,
            chain = chain,
            network = network
        }
    end
    
    if not response.success then
        ngx.log(ngx.ERR, "[CACHE_GET] Cache service returned error: ", response.error or "unknown")
        return {
            cache_type = nil,
            cache_key = nil,
            ttl = nil,
            cached_response = nil,
            raw_body = body_data,
            chain = chain,
            network = network
        }
    end
    
    local cached_response = nil
    local cache_status = response.cache_status or "MISS"
    local cache_level = response.cache_level or "MISS"
    
    if response.found and response.data then
        cached_response = response.data
    end
    
    return {
        cache_type = response.cache_type,
        cache_key = response.key,
        ttl = nil, -- cache will calculate ttl itself
        cached_response = cached_response,
        raw_body = body_data,
        fresh = response.fresh,
        cache_status = cache_status,
        cache_level = cache_level,
        chain = chain,
        network = network
    }
end

-- Save response to cache
function _M.save_to_cache(cache_info, response_body)
    if not cache_info.cache_key or not cache_info.raw_body then
        ngx.log(ngx.WARN, "[CACHE_SET] Missing cache info parameters, cannot save to cache")
        return false
    end
    
    if not cache_info.cache_type then
        return false
    end
    
    -- Prepare request for cache service
    local cache_request = {
        chain = cache_info.chain,
        network = cache_info.network,
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
        ngx.log(ngx.ERR, "[CACHE_SET] Failed to save to cache service: ", err)
        return false
    end
    
    if not response.success then
        ngx.log(ngx.ERR, "[CACHE_SET] Cache service save failed: ", response.error or "unknown")
        return false
    end
    return true
end

-- Function to reset cache instances (for testing compatibility)
function _M.reset_cache_instances()
    -- No-op for HTTP-based cache
    ngx.log(ngx.INFO, "[CACHE_DEBUG] reset_cache_instances called (no-op for HTTP cache)")
end

-- Function to run cache diagnostics
function _M.run_diagnostics()
    cache_diagnostics.test_cache_connectivity()
end

return _M