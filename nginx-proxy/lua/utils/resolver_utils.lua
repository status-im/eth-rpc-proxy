local _M = {}

local function resolve_url_with_custom_dns(url, custom_dns)
    ngx.log(ngx.INFO, "resolver_utils: resolve_url_with_custom_dns() called with URL: '", 
            tostring(url), "', DNS: '", tostring(custom_dns), "'")
    
    -- Validate inputs
    if not url or url == "" then
        ngx.log(ngx.ERR, "resolver_utils: URL is required")
        return nil, "URL is required"
    end
    
    if not custom_dns or custom_dns == "" then
        ngx.log(ngx.ERR, "resolver_utils: Custom DNS resolver is required")
        return nil, "Custom DNS resolver is required"
    end

    -- Parse URL using regex (resty.http doesn't support redis:// scheme)
    ngx.log(ngx.INFO, "resolver_utils: Attempting to parse URL: ", url)
    
    -- Parse URL: scheme://host:port/path?query
    local scheme, host, port, path_and_query = url:match("^(%w+)://([^:/]+):?(%d*)(.*)$")
    if not scheme or not host then
        ngx.log(ngx.ERR, "resolver_utils: Failed to parse URL: ", url)
        return nil, "Invalid URL format - expected scheme://host:port[/path][?query]"
    end
    
    -- Extract path and query from path_and_query
    local path, query = "", ""
    if path_and_query and path_and_query ~= "" then
        local query_start = path_and_query:find("?")
        if query_start then
            path = path_and_query:sub(1, query_start - 1)
            query = path_and_query:sub(query_start + 1)
        else
            path = path_and_query
        end
    end
    
    -- Set default port if not specified
    if not port or port == "" then
        if scheme == "redis" then
            port = "6379"
        elseif scheme == "http" then
            port = "80"
        elseif scheme == "https" then
            port = "443"
        else
            port = "80"  -- fallback
        end
    end
    
    ngx.log(ngx.INFO, "resolver_utils: Parsed URL - scheme: ", tostring(scheme), 
            ", host: ", tostring(host), ", port: ", tostring(port), 
            ", path: ", tostring(path), ", query: ", tostring(query))

    ngx.log(ngx.INFO, "resolver_utils: Resolving host: ", host, " using DNS: ", custom_dns)

    -- Initialize DNS resolver
    ngx.log(ngx.INFO, "resolver_utils: Initializing DNS resolver with nameserver: ", custom_dns)
    local dns = require("resty.dns.resolver")
    local r, err = dns:new({
        nameservers = {custom_dns},
        retrans = 5,  -- 5 retransmissions on receive timeout
        timeout = 2000,  -- 2 sec
    })

    if not r then
        ngx.log(ngx.ERR, "resolver_utils: Failed to initialize DNS resolver: ", err or "unknown error")
        return nil, "DNS resolver initialization failed"
    end
    
    ngx.log(ngx.INFO, "resolver_utils: DNS resolver initialized successfully")

    -- Perform DNS resolution
    ngx.log(ngx.INFO, "resolver_utils: Performing DNS query for host: ", host)
    local answers, err = r:query(host)
    if not answers then
        ngx.log(ngx.ERR, "resolver_utils: DNS resolution failed for host '", host, "': ", err or "unknown error")
        return nil, "DNS resolution failed"
    end
    
    ngx.log(ngx.INFO, "resolver_utils: DNS query completed, processing answers")

    -- Process DNS answers
    if answers.errcode then
        ngx.log(ngx.ERR, "resolver_utils: DNS server returned error code: ", answers.errcode, 
                " ", answers.errstr or "unknown error")
        return nil, "DNS server error"
    end

    ngx.log(ngx.INFO, "resolver_utils: Processing ", #answers, " DNS answers")

    -- Return complete URL with resolved IP using parsed components
    for i, ans in ipairs(answers) do
        if ans.address then
            ngx.log(ngx.INFO, "resolver_utils: Found IP address: ", ans.address, " for host: ", host)
            -- Reconstruct URL using parsed components
            local resolved_url = scheme .. "://" .. ans.address .. ":" .. port
            if path and path ~= "" then
                resolved_url = resolved_url .. path
            end
            if query and query ~= "" then
                resolved_url = resolved_url .. "?" .. query
            end
            ngx.log(ngx.INFO, "resolver_utils: Successfully constructed resolved URL: ", resolved_url)
            return resolved_url
        else
            ngx.log(ngx.WARN, "resolver_utils: DNS answer ", i, " has no address field")
        end
    end

    ngx.log(ngx.ERR, "resolver_utils: No IP addresses found for host: ", host, " in ", #answers, " DNS answers")
    return nil, "No IP addresses found"
end

_M.resolve_url_with_custom_dns = resolve_url_with_custom_dns

return _M
