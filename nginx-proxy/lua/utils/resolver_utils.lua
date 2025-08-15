local _M = {}

local function resolve_url_with_custom_dns(url, custom_dns)
    -- Validate inputs
    if not url or url == "" then
        ngx.log(ngx.ERR, "URL is required")
        return nil, "URL is required"
    end

    if not custom_dns or custom_dns == "" then
        ngx.log(ngx.ERR, "resolver_utils: Custom DNS resolver is required")
        return nil, "Custom DNS resolver is required"
    end

    -- Parse URL
    local url_parts, err = parse_url(url)
    if not url_parts then
        ngx.log(ngx.ERR, "resolver_utils: Failed to parse URL: ", err)
        return nil, err
    end

    -- Initialize DNS resolver
    local dns = require("resty.dns.resolver")
    local r, err = dns:new({
        nameservers = {custom_dns},
        retrans = 5,
        timeout = 2000,
    })

    if not r then
        ngx.log(ngx.ERR, "resolver_utils: Failed to initialize DNS resolver: ", err or "unknown error")
        return nil, "DNS resolver initialization failed"
    end

    -- Perform DNS resolution
    local answers, err = r:query(url_parts.host)
    if not answers then
        ngx.log(ngx.ERR, "resolver_utils: DNS resolution failed for host '", url_parts.host, "': ", err or "unknown error")
        return nil, "DNS resolution failed"
    end

    -- Process DNS answers
    if answers.errcode then
        ngx.log(ngx.ERR, "resolver_utils: DNS server returned error code: ", answers.errcode, 
                " ", answers.errstr or "unknown error")
        return nil, "DNS server error"
    end

    -- Return complete URL with resolved IP
    for i, ans in ipairs(answers) do
        if ans.address then
            ngx.log(ngx.INFO, "Resolved IP: ", ans.address)
            -- Reconstruct URL using parsed components
            local resolved_url = url_parts.scheme .. "://" .. ans.address
            if url_parts.port then
                resolved_url = resolved_url .. ":" .. url_parts.port
            end
            if url_parts.path then
                resolved_url = resolved_url .. url_parts.path
            end
            if url_parts.query and url_parts.query ~= "" then
                resolved_url = resolved_url .. "?" .. url_parts.query
            end
            return resolved_url
        end
    end

    ngx.log(ngx.ERR, "resolver_utils: No IP addresses found for host: ", url_parts.host)
    return nil, "No IP addresses found"
end


-- Parse URL into components: scheme, host, port, path, query
local function parse_url(url)
    if not url or url == "" then
        return nil, "URL is required"
    end

    -- Parse URL: scheme://host:port/path?query
    local scheme, host, port, path_and_query = url:match("^(%w+)://([^:/?]+):?(%d*)(.*)$")
    if not scheme or not host then
        return nil, "Invalid URL format - expected scheme://host:port[/path][?query]"
    end

    -- Extract path and query from path_and_query
    local path, query = "", ""
    if path_and_query and path_and_query ~= "" then
        local query_start = path_and_query:find("?")
        if query_start then
            if query_start == 1 then
                -- URL like http://example.com?query (no path, only query)
                path = ""
                query = path_and_query:sub(2)
            else
                -- URL like http://example.com/path?query
                path = path_and_query:sub(1, query_start - 1)
                query = path_and_query:sub(query_start + 1)
            end
        else
            path = path_and_query
        end
    end

    -- Set default port if not specified
    if not port or port == "" then
        if scheme == "redis" then
            port = "6379"
        elseif scheme == "rediss" then
            port = "6380"
        elseif scheme == "http" then
            port = "80"
        elseif scheme == "https" then
            port = "443"
        else
            port = "80"
        end
    end

    return {
        scheme = scheme,
        host = host,
        port = port,
        path = path,
        query = query
    }
end

_M.parse_url = parse_url
_M.resolve_url_with_custom_dns = resolve_url_with_custom_dns

return _M
