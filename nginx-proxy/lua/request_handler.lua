local json = require("cjson")
local http = require("resty.http")
local cache = require("cache.cache")

-- HTTP status codes that should trigger retry
local retry_status = {
    [401] = true, [402] = true, [403] = true, [429] = true,
    [500] = true, [501] = true, [502] = true, [503] = true, [504] = true, [505] = true
}

-- JSON-RPC error codes that should trigger retry
local retry_evm = {
    [32005] = true, -- Request timeout
    [33000] = true, -- Rate limit exceeded
    [33300] = true, -- Network error
    [33400] = true  -- Provider error
}

-- Check if we should retry with next provider
local function should_retry(res, decoded)
    if not res then return true end
    if retry_status[res.status] then return true end
    if decoded and decoded.error and retry_evm[decoded.error.code] then return true end
    return false
end

-- Filter providers by provider_type if specified
local function filter_providers(providers, provider_type)
    local providers_to_try = {}
    local tried_specific_provider = false

    if provider_type and provider_type ~= "" then
        for _, provider in ipairs(providers) do
            if provider.type == provider_type then
                table.insert(providers_to_try, provider)
            end
        end
        tried_specific_provider = true
    else
        providers_to_try = providers
    end

    return providers_to_try, tried_specific_provider
end

-- Read request body once and reuse it throughout the handler
ngx.req.read_body()
local body_data = ngx.req.get_body_data() or ""

-- Extract and validate path parameters
local chain, network, provider_type = ngx.var.uri:match("^/([^/]+)/([^/]+)/?([^/]*)$")
if not chain or not network then
    ngx.log(ngx.ERR, "Invalid URL format - must be /chain/network or /chain/network/provider_type")
    ngx.status = 400
    ngx.say("Invalid URL format - must be /chain/network or /chain/network/provider_type")
    return
end

-- Get providers for the requested chain/network
local chain_network_key = chain .. ":" .. network
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
local providers_to_try, tried_specific_provider = filter_providers(providers, provider_type)

if #providers_to_try == 0 then
    ngx.log(ngx.ERR, "No providers found for provider_type: ", provider_type or "none")
    ngx.status = 404
    ngx.say("No providers available for this provider type")
    return
end

local success = false

for _, provider in ipairs(providers_to_try) do
    local httpc = http.new()

    -- Handle authentication based on provider config
    local request_url = provider.url
    local request_headers = {
        ["Content-Type"] = "application/json"
    }

    if provider.authType == "token-auth" and provider.authToken then
        if request_url:sub(-1) == "/" then
            request_url = request_url .. provider.authToken
        else
            request_url = request_url .. "/" .. provider.authToken
        end
    elseif provider.authType == "basic-auth" and provider.authLogin and provider.authPassword then
        local auth_str = ngx.encode_base64(provider.authLogin .. ":" .. provider.authPassword)
        request_headers["Authorization"] = "Basic " .. auth_str
    end

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

        -- Check if we should retry with next provider
        if not should_retry(res, decoded) then
            -- Cache response if cacheable
            if cache_info.cache_type then
                cache.save_to_cache(cache_info, res.body)
            end

            -- Success! Use proxy_response to handle headers and body automatically
            httpc:proxy_response(res)
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