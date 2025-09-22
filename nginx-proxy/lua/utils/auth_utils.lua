local _M = {}

-- Extract JWT token from various sources (Authorization header or query parameter)
function _M.extract_jwt_token()
    -- First, try to get token from Authorization header (existing behavior)
    local auth_header = ngx.var.http_authorization
    if auth_header then
        local auth_type, token = auth_header:match("^(%S+)%s+(.+)$")
        if auth_type == "Bearer" and token then
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
                    args[key] = value ~= "" and value or true
                end
            end
        end
    end
    
    -- Check for 'token' parameter
    if args.token then
        return args.token, "query"
    end
    
    -- Check for 'jwt' parameter  
    if args.jwt then
        return args.jwt, "query"
    end
    
    -- Check for 'access_token' parameter (common OAuth2 pattern)
    if args.access_token then
        return args.access_token, "query"
    end
    
    return nil, nil
end


return _M
