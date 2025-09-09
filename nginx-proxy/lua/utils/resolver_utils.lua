local _M = {}

local function resolve_url_with_custom_dns(url, custom_dns)
    -- Validate inputs
    if not url or url == "" then
        ngx.log(ngx.ERR, "URL is required")
        return nil, "URL is required"
    end

    if not custom_dns or custom_dns == "" then
        ngx.log(ngx.ERR, "Custom DNS resolver is required")
        return nil, "Custom DNS resolver is required"
    end

    -- Parse URL using resty.http
    local http = require("resty.http")
    local scheme, host, port, path, query = unpack(http:parse_uri(url))
    if not scheme then
        ngx.log(ngx.ERR, "Failed to parse URL: ", url)
        return nil, "Invalid URL format"
    end

    ngx.log(ngx.INFO, "Resolving host: ", host, " using DNS: ", custom_dns)

    -- Initialize DNS resolver
    local dns = require("resty.dns.resolver")
    local r, err = dns:new({
        nameservers = {custom_dns},
        retrans = 5,  -- 5 retransmissions on receive timeout
        timeout = 2000,  -- 2 sec
    })

    if not r then
        ngx.log(ngx.ERR, "Failed to initialize DNS resolver: ", err)
        return nil, "DNS resolver initialization failed"
    end

    -- Perform DNS resolution
    local answers, err = r:query(host)
    if not answers then
        ngx.log(ngx.ERR, "DNS resolution failed: ", err)
        return nil, "DNS resolution failed"
    end

    -- Process DNS answers
    if answers.errcode then
        ngx.log(ngx.ERR, "DNS server returned error code: ", answers.errcode,
                " ", answers.errstr)
        return nil, "DNS server error"
    end

    -- Return complete URL with resolved IP using parsed components
    for i, ans in ipairs(answers) do
        if ans.address then
            ngx.log(ngx.INFO, "Resolved IP: ", ans.address)
            -- Reconstruct URL using parsed components
            local resolved_url = scheme .. "://" .. ans.address
            if port then
                resolved_url = resolved_url .. ":" .. port
            end
            if path then
                resolved_url = resolved_url .. path
            end
            if query and query ~= "" then
                resolved_url = resolved_url .. "?" .. query
            end
            return resolved_url
        end
    end

    ngx.log(ngx.ERR, "No IP addresses found for host: ", host)
    return nil, "No IP addresses found"
end

_M.resolve_url_with_custom_dns = resolve_url_with_custom_dns

return _M
