local json = require("cjson")
local http = require("resty.http")

-- Extract and validate path parameters
local chain, network, provider_type = ngx.var.uri:match("^/([^/]+)/([^/]+)/?([^/]*)$")
if not chain or not network then
    ngx.log(ngx.ERR, "Invalid URL format - must be /chain/network or /chain/network/provider_type")
    ngx.status = 400
    ngx.say("Invalid URL format - must be /chain/network or /chain/network/provider_type")
    return
end

ngx.log(ngx.INFO, "Chain: ", chain, " Network: ", network, provider_type and (" Provider: " .. provider_type) or "")

-- Get providers for the requested chain/network
local chain_network_key = chain .. ":" .. network
local providers_str = ngx.shared.providers:get(chain_network_key)

if not providers_str then
    ngx.log(ngx.ERR, "No providers found for ", chain_network_key)
    ngx.status = 404
    ngx.say("No providers available for this chain/network")
    return
end
local providers = json.decode(providers_str)
local body_data = ngx.req.get_body_data()
ngx.log(ngx.INFO, "Request body: ", body_data)

if #providers == 0 then
    ngx.log(ngx.ERR, "No providers found for ", chain_network_key)
    ngx.status = 404
    ngx.say("No providers available for this chain/network")
    return
end

local tried_specific_provider = false
for _, provider in ipairs(providers) do
    -- Skip providers that don't match requested provider_type
    if provider_type and provider_type ~= "" then
        if provider.type ~= provider_type then
            goto continue
        end
        tried_specific_provider = true
    end

    ngx.log(ngx.INFO, "provider: ", provider.url)
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
        body = ngx.req.get_body_data(),
        headers = request_headers,
        ssl_verify = false,
        options = { family = ngx.AF_INET }
    })

    if not res then
        ngx.log(ngx.ERR, "HTTP request failed: ", err)
    else
        ngx.log(ngx.DEBUG, "Response body: ", res.body)
        if res.status == 401 or res.status == 402 or res.status == 403 or res.status == 429 or
           (res.status >= 500 and res.status < 600) then
            ngx.log(ngx.ERR, "Error status ", res.status, ", trying next provider")
            goto continue
        end

        local ok, decoded_body = pcall(json.decode, res.body)
        if ok and decoded_body.error and decoded_body.error.code then
            if decoded_body.error.code == 32005 or
               decoded_body.error.code == 33000 or
               decoded_body.error.code == 33300 or
               decoded_body.error.code == 33400 then
                ngx.log(ngx.ERR, "JSON error code ", decoded_body.error.code, ", trying next provider")
                goto continue
            end
        end

        -- Set Content-Type header
        if res.headers["Content-Type"] then
            ngx.header["Content-Type"] = res.headers["Content-Type"]
        end

        -- Set Content-Length header if present
        if res.headers["Content-Length"] then
            ngx.header["Content-Length"] = res.headers["Content-Length"]
        end

        -- Set Vary header if present
        if res.headers["Vary"] then
            ngx.header["Vary"] = res.headers["Vary"]
        end

        ngx.say(res.body)
        return
    end

    ::continue::
end

if provider_type and provider_type ~= "" and not tried_specific_provider then
    ngx.status = 404
    ngx.say("Provider not found: " .. provider_type)
else
    ngx.status = 502
    ngx.say("All providers failed")
end