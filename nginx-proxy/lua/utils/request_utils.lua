local _M = {}

-- HTTP status codes that should trigger retry
_M.RETRY_STATUS = {
    [401] = true, [402] = true, [403] = true, [429] = true,
    [500] = true, [501] = true, [502] = true, [503] = true, [504] = true, [505] = true
}

-- JSON-RPC error codes that should trigger retry
_M.RETRY_EVM = {
    [32005] = true, -- Request timeout
    [33000] = true, -- Rate limit exceeded
    [33300] = true, -- Network error
    [33400] = true  -- Provider error
}

-- Headers that should not be proxied to client (hop-by-hop and CORS)
_M.HEADER_BLOCKLIST = {
    -- Connection-related headers (hop-by-hop)
    ["connection"] = true,
    ["transfer-encoding"] = true,
    ["keep-alive"] = true,
    ["upgrade"] = true,
    ["proxy-authenticate"] = true,
    ["proxy-authorization"] = true,
    ["te"] = true,
    ["trailer"] = true,
    
    -- CORS headers (we control these ourselves)
    ["access-control-allow-origin"] = true,
    ["access-control-allow-credentials"] = true,
    ["access-control-allow-methods"] = true,
    ["access-control-allow-headers"] = true,
    ["access-control-expose-headers"] = true,
    ["access-control-max-age"] = true,
}

-- Check if we should retry with next provider
function _M.should_retry(res, decoded)
    if not res then return true end
    if _M.RETRY_STATUS[res.status] then return true end
    if decoded and decoded.error and _M.RETRY_EVM[decoded.error.code] then return true end
    return false
end

-- Filter providers by provider_type if specified
function _M.filter_providers(providers, provider_type)
    if not providers or type(providers) ~= "table" then
        return {}, false
    end
    
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

-- Parse and validate URL path parameters  
function _M.parse_url_path(uri)
    if not uri or type(uri) ~= "string" then
        return nil, nil, nil, "Invalid URI"
    end
    
    local chain, network, provider_type = uri:match("^/([^/]+)/([^/]+)/?([^/]*)$")
    if not chain or not network then
        return nil, nil, nil, "Invalid URL format - must be /chain/network or /chain/network/provider_type"
    end
    
    -- Convert empty provider_type to nil for consistency
    if provider_type == "" then
        provider_type = nil
    end
    
    return chain, network, provider_type, nil
end

-- Setup authentication for provider request
function _M.setup_auth(provider, base_url, base_headers)
    if not provider or type(provider) ~= "table" then
        return base_url or "", base_headers or {}
    end
    
    local request_url = provider.url or base_url or ""
    local request_headers = {}
    
    -- Copy base headers
    if base_headers then
        for k, v in pairs(base_headers) do
            request_headers[k] = v
        end
    end
    
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
    
    return request_url, request_headers
end

-- Filter response headers based on blocklist
function _M.filter_response_headers(headers)
    if not headers or type(headers) ~= "table" then
        return {}
    end
    
    local filtered_headers = {}
    for key, value in pairs(headers) do
        local lower_key = string.lower(key)
        if not _M.HEADER_BLOCKLIST[lower_key] then
            filtered_headers[key] = value
        end
    end
    
    return filtered_headers
end

-- Create chain:network key for provider lookup
function _M.get_chain_network_key(chain, network)
    if not chain or not network then
        return nil
    end
    return chain .. ":" .. network
end

return _M 