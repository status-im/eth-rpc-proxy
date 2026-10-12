local json = require("cjson")
local http = require("resty.http")
local cache = require("cache.cache")
local request_utils = require("utils.request_utils")
local request_body = require("utils.request_body")
local route = require("utils.route")

-- Answers a request that cannot be served
local function fail(failure)
    if failure.log then
        ngx.log(ngx.ERR, failure.log)
    end
    ngx.status = failure.status
    ngx.say(failure.message)
end

-- Read request body once and reuse it throughout the handler
ngx.req.read_body()
local wire_body = ngx.req.get_body_data() or ""
local body_data, decode_status, decode_err = request_body.decode(wire_body, ngx.var.http_content_encoding)
if not body_data then
    ngx.log(ngx.WARN, decode_err)
    ngx.status = decode_status
    ngx.say(decode_err)
    return
end
if body_data ~= wire_body then
    local stats = ngx.shared.stats
    stats:incr("requests_gzip", 1, 0)
    stats:incr("request_bytes_gzip_wire", #wire_body, 0)
    stats:incr("request_bytes_gzip_decoded", #body_data, 0)
end

-- Route by the path to the providers stored for its chain/network
local target, route_failure = route.target(ngx.var.uri)
if not target then
    return fail(route_failure)
end
local provider_type = target.provider_type

local providers
providers, route_failure = route.providers(ngx.shared.providers:get(target.key), target.key)
if not providers then
    return fail(route_failure)
end

-- Check cache with unified function (handles all cache operations)
local cache_info = cache.check_cache(target.chain, target.network, body_data)
if cache_info.cached_response then
    ngx.header["Content-Type"] = "application/json"
    if cache_info.cache_status then
        ngx.header["X-Cache-Status"] = cache_info.cache_status
    end
    if cache_info.cache_level then
        ngx.header["X-Cache-Level"] = cache_info.cache_level
    end
    ngx.say(cache_info.cached_response)
    return
end

local candidates
candidates, route_failure = route.candidates(providers, provider_type, target.key)
if not candidates then
    return fail(route_failure)
end

local success = false

for _, provider in ipairs(candidates.providers) do
    local httpc = http.new()

    -- Setup authentication and headers using request_utils
    local base_headers = {
        ["Content-Type"] = "application/json"
    }
    local request_url, request_headers = request_utils.setup_auth(provider, provider.url, base_headers)

    local res, err = httpc:request_uri(request_url, {
        method = ngx.req.get_method(),
        body = body_data,
        headers = request_headers,
        ssl_verify = false,
        options = { family = ngx.AF_INET }
    })

    if res then
        local ok, decoded_response = pcall(json.decode, res.body)
        local decoded = ok and decoded_response or nil

        -- Check if we should retry with next provider using request_utils
        if not request_utils.should_retry(res, decoded) then
            -- Cache response if cacheable
            if cache_info.cache_type then
                cache.save_to_cache(cache_info, res.body)
            end

            -- Success! Manually set response, filtering out unwanted headers
            ngx.status = res.status
            
            -- Filter headers using request_utils
            local filtered_headers = request_utils.filter_response_headers(res.headers)
            for key, value in pairs(filtered_headers) do
                ngx.header[key] = value
            end
            
            -- Set cache status header for non-cached responses
            if cache_info.cache_status then
                ngx.header["X-Cache-Status"] = cache_info.cache_status
            else
                -- Default to MISS if cache_status is not available
                ngx.header["X-Cache-Status"] = "MISS"
            end
            
            -- Set cache level header for non-cached responses
            if cache_info.cache_level then
                ngx.header["X-Cache-Level"] = cache_info.cache_level
            else
                -- Default to MISS if cache_level is not available
                ngx.header["X-Cache-Level"] = "MISS"
            end
            
            -- Set response body
            ngx.say(res.body)
            success = true
            break
        else
            if res.status then
                ngx.log(ngx.ERR, "Error status ", res.status, ", trying next provider")
            elseif decoded and decoded.error then
                ngx.log(ngx.ERR, "JSON-RPC error code ", decoded.error.code, ", trying next provider")
            end
        end
    else
        ngx.log(ngx.ERR, "HTTP request failed: ", err)
    end
end

if not success then
    fail(route.exhausted(provider_type, candidates.tried_specific))
end
