local json = require("cjson")
local http = require("resty.http")

-- Extract and validate path parameters
local chain, network, provider_id = ngx.var.uri:match("^/([^/]+)/([^/]+)/?([^/]*)$")
if not chain or not network then
    ngx.log(ngx.ERR, "Invalid URL format - must be /chain/network or /chain/network/provider_id")
    ngx.status = 400
    ngx.say("Invalid URL format - must be /chain/network or /chain/network/provider_id")
    return
end

ngx.log(ngx.INFO, "Chain: ", chain, " Network: ", network, provider_id and (" Provider: " .. provider_id) or "")

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
    -- Skip providers that don't match requested provider_id
    if provider_id and provider_id ~= "" then
        if provider.id ~= provider_id then
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
        ngx.log(ngx.INFO, "Response body: ", res.body)
        local ok, decoded_body = pcall(json.decode, res.body)
        if ok then
            ngx.say(json.encode(decoded_body))
            return
        else
            ngx.log(ngx.ERR, "Failed to decode response: ", decoded_body)
        end
    end

    ::continue::
end

if provider_id and provider_id ~= "" and not tried_specific_provider then
    ngx.status = 404
    ngx.say("Provider not found: " .. provider_id)
else
    ngx.status = 502
    ngx.say("All providers failed")
end 