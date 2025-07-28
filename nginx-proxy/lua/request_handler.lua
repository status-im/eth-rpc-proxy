local json = require("cjson")
local http = require("resty.http")
local cache = require("cache.cache")
local request_utils = require("utils.request_utils")

-- Read request body once and reuse it throughout the handler
ngx.req.read_body()
local body_data = ngx.req.get_body_data() or ""

-- Extract and validate path parameters
local chain, network, provider_type, err = request_utils.parse_url_path(ngx.var.uri)
if err then
    ngx.log(ngx.ERR, err)
    ngx.status = 400
    ngx.say(err)
    return
end

-- Get providers for the requested chain/network
local chain_network_key = request_utils.get_chain_network_key(chain, network)
local providers_str = ngx.shared.providers:get(chain_network_key)

if not providers_str then
    ngx.log(ngx.ERR, "No providers found for ", chain_network_key)
    ngx.status = 404
    ngx.say("No providers available for this chain/network")
    return
end

-- Safely decode providers JSON with error handling
local ok, providers = pcall(json.decode, providers_str)
if not ok then
    ngx.log(ngx.ERR, "Invalid providers JSON for ", chain_network_key, ": ", providers)
    ngx.status = 500
    ngx.say("Internal server error: invalid providers configuration")
    return
end

-- Check cache with unified function (handles all cache operations)
local cache_info = cache.check_cache(chain, network, body_data)
if cache_info.cached_response then
    ngx.header["Content-Type"] = "application/json"
    ngx.say(cache_info.cached_response)
    return
end

if #providers == 0 then
    ngx.log(ngx.ERR, "No providers found for ", chain_network_key)
    ngx.status = 404
    ngx.say("No providers available for this chain/network")
    return
end

-- Filter providers by provider_type if specified
local providers_to_try, tried_specific_provider = request_utils.filter_providers(providers, provider_type)

if #providers_to_try == 0 then
    ngx.log(ngx.ERR, "No providers found for provider_type: ", provider_type or "none")
    ngx.status = 404
    ngx.say("No providers available for this provider type")
    return
end

local success = false

for _, provider in ipairs(providers_to_try) do
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
    if provider_type and provider_type ~= "" and not tried_specific_provider then
        ngx.status = 404
        ngx.say("Provider not found: " .. provider_type)
    else
        ngx.status = 502
        ngx.say("All providers failed")
    end
end