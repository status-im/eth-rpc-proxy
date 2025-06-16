local json = require("cjson")

local _M = {}

-- Base64 URL decode function
local function base64_url_decode(str)
    -- Add padding if needed
    local padding = 4 - (#str % 4)
    if padding ~= 4 then
        str = str .. string.rep("=", padding)
    end
    
    -- Replace URL-safe characters
    str = str:gsub("-", "+"):gsub("_", "/")
    
    return ngx.decode_base64(str)
end

-- Parse and validate JWT token (simplified - no signature verification)
function _M.validate_token(token)
    if not token then
        return false, "No token provided"
    end
    
    -- Split token into parts
    local parts = {}
    for part in token:gmatch("[^%.]+") do
        table.insert(parts, part)
    end
    
    if #parts ~= 3 then
        return false, "Invalid token format"
    end
    
    local header_b64, payload_b64, signature_b64 = parts[1], parts[2], parts[3]
    
    -- Decode header and payload
    local header_json = base64_url_decode(header_b64)
    local payload_json = base64_url_decode(payload_b64)
    
    if not header_json or not payload_json then
        return false, "Invalid token encoding"
    end
    
    -- Parse JSON
    local ok, header = pcall(json.decode, header_json)
    if not ok then
        return false, "Invalid header JSON"
    end
    
    local ok, payload = pcall(json.decode, payload_json)
    if not ok then
        return false, "Invalid payload JSON"
    end
    
    -- Check algorithm
    if header.alg ~= "HS256" then
        return false, "Unsupported algorithm: " .. (header.alg or "none")
    end
    
    -- NOTE: Signature verification skipped - trusting Go service validation
    
    -- Check expiration
    local now = ngx.time()
    if payload.exp and payload.exp < now then
        return false, "Token expired"
    end
    
    -- Check not before
    if payload.nbf and payload.nbf > now then
        return false, "Token not yet valid"
    end
    
    return true, payload
end

-- Check token usage limit
function _M.check_usage_limit(token_id, current_count, max_requests)
    if not token_id or not max_requests then
        return true -- No limit checking
    end
    
    if current_count >= max_requests then
        return false, "Request limit exceeded"
    end
    
    return true
end

-- Increment token usage counter
function _M.increment_usage(token_id)
    if not token_id then
        return 0
    end
    
    local key = "usage:" .. token_id
    local current = ngx.shared.jwt_tokens:get(key) or 0
    local new_count = current + 1
    
    -- Store with 1 hour expiration
    ngx.shared.jwt_tokens:set(key, new_count, 3600)
    
    return new_count
end

-- Get current token usage
function _M.get_usage(token_id)
    if not token_id then
        return 0
    end
    
    local key = "usage:" .. token_id
    return ngx.shared.jwt_tokens:get(key) or 0
end

return _M 