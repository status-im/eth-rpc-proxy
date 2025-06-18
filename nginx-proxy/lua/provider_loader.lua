local http = require("resty.http")
local json = require("cjson")
local resolver_utils = require("resolver_utils")

local M = {}
local function read_config_from_url(url)
    if not url or url == "" then
        return nil, "URL is invalid or not provided"
    end

    local httpc = http.new()
    local custom_dns = os.getenv("CUSTOM_DNS")
    local request_url = url
    
    -- Resolve URL using custom DNS if available
    if custom_dns and custom_dns ~= "" then
        local resolved_url, resolve_err = resolver_utils.resolve_url_with_custom_dns(url, custom_dns)
        if not resolved_url then
            ngx.log(ngx.ERR, "Failed to resolve URL with custom DNS: ", resolve_err)
            request_url = url  -- Fall back to original URL
        else
            request_url = resolved_url
        end
    end

    local res, err = httpc:request_uri(request_url, {
        method = "GET",
        headers = {
            ["Content-Type"] = "application/json",
        },
        ssl_verify = false,
    })

    if not res then
        ngx.log(ngx.ERR, "Failed to fetch configuration from URL: ", err)
        return nil, err
    end

    if res.status ~= 200 then
        ngx.log(ngx.ERR, "Non-200 response from URL: ", res.status)
        return nil, "HTTP error: " .. res.status
    end

    ngx.log(ngx.INFO, "Successfully fetched configuration from URL: ", request_url)
    return res.body, nil
end

-- Function to read configuration from file
local function read_config_from_file(filepath)
    ngx.log(ngx.INFO, "Reading configuration from file: ", filepath)
    if not filepath or filepath == "" then
        return nil, "Filepath is invalid or not provided"
    end

    local file = io.open(filepath, "r")
    if not file then
        ngx.log(ngx.ERR, "Failed to open file: ", filepath)
        return nil, "File open error"
    end

    local content = file:read("*all")
    file:close()
    ngx.log(ngx.INFO, "Successfully read configuration from file")
    return content, nil
end

-- Main provider reload function
function M.reload_providers(premature, url, fallbackLocalConfig)
    if premature then
        return
    end

    ngx.log(ngx.INFO, "Reloading providers")

    -- Attempt to load configuration from URL
    local config, err = read_config_from_url(url)
    if not config then
        ngx.log(ngx.ERR, "Failed to load configuration from URL: ", err)

        -- Attempt to load configuration from file
        local file_config, file_err = read_config_from_file(fallbackLocalConfig)
        if not file_config then
            ngx.log(ngx.ERR, "Failed to load configuration from fallback file: ", file_err)
            return
        end
        config = file_config
    end

    -- Parse and transform provider configuration
    local parsed_config, parse_err = json.decode(config)
    if not parsed_config then
        ngx.log(ngx.ERR, "Failed to parse provider config: ", parse_err)
        return
    end

    -- Clear existing providers
    ngx.shared.providers:flush_all()

    -- Store providers by chain/network
    for _, chain in ipairs(parsed_config.chains or {}) do
        local key = chain.name .. ":" .. chain.network
        ngx.shared.providers:set(key, json.encode(chain.providers))
        ngx.log(ngx.INFO, "Loaded providers for ", key, json.encode(chain.providers))
    end

    ngx.log(ngx.INFO, "Providers reloaded and stored by chain/network")
end

-- Scheduler to call reload_providers
function M.schedule_reload_providers(url, fallbackLocalConfig)
    local ok, err = ngx.timer.at(0, M.reload_providers, url, fallbackLocalConfig)
    if not ok then
        ngx.log(ngx.ERR, "Failed to create timer: ", err)
    end
end

return M
