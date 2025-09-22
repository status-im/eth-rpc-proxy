local _M = {}

-- Extract JWT token from various sources (Authorization header or query parameter)
function _M.extract_jwt_token()
    -- First, try to get token from Authorization header (existing behavior)
    local auth_header = ngx.var.http_authorization
    if auth_header then
        local auth_type, token = auth_header:match("^(%S+)%s+(.+)$")
        if auth_type == "Bearer" and token then
            -- Validate token length to prevent memory issues (JWT tokens are typically < 2KB)
            if #token > 4096 then
                ngx.log(ngx.WARN, "Token too long, rejecting: ", #token, " bytes")
                return nil, nil
            end
            return token, "header"
        end
    end
    
    -- If no valid Authorization header, try query parameters
    local args = ngx.req.get_uri_args()
    
    -- For auth_request subrequests, also check parent request URI for query params
    local request_uri = ngx.var.request_uri
    if request_uri then
        local query_start = request_uri:find("?")
        if query_start then
            local query_string = request_uri:sub(query_start + 1)
            -- Parse query string manually
            for pair in string.gmatch(query_string, "[^&]+") do
                local key, value = pair:match("([^=]+)=?(.*)")
                if key then
                    -- URL decode the key as well for completeness
                    key = ngx.unescape_uri(key)
                    if value ~= "" then
                        -- URL decode the value to handle JWT tokens with encoded characters
                        args[key] = ngx.unescape_uri(value)
                    else
                        args[key] = true
                    end
                end
            end
        end
    end
    
    -- Check for 'token' parameter
    if args.token then
        -- Validate token length to prevent memory issues
        if #args.token > 4096 then
            ngx.log(ngx.WARN, "Query token too long, rejecting: ", #args.token, " bytes")
            return nil, nil
        end
        return args.token, "query"
    end
    
    -- Check for 'jwt' parameter  
    if args.jwt then
        -- Validate token length to prevent memory issues
        if #args.jwt > 4096 then
            ngx.log(ngx.WARN, "Query JWT too long, rejecting: ", #args.jwt, " bytes")
            return nil, nil
        end
        return args.jwt, "query"
    end
    
    -- Check for 'access_token' parameter (common OAuth2 pattern)
    if args.access_token then
        -- Validate token length to prevent memory issues
        if #args.access_token > 4096 then
            ngx.log(ngx.WARN, "Query access_token too long, rejecting: ", #args.access_token, " bytes")
            return nil, nil
        end
        return args.access_token, "query"
    end
    
    return nil, nil
end


return _M
