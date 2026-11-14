local cjson = require("cjson")

local _M = {}

-- Claims of a JWT token, or nil when it is not shaped like one.
--
-- The signature is NOT checked here: use it only for a token the auth service
-- has already verified.
function _M.decode(token)
    local payload = token:match("^[^.]+%.([^.]+)%.[^.]+$")
    if not payload then
        return nil
    end

    -- base64url without padding -> base64
    payload = payload:gsub("-", "+"):gsub("_", "/")
    local tail = #payload % 4
    if tail == 1 then
        return nil
    elseif tail > 0 then
        payload = payload .. string.rep("=", 4 - tail)
    end

    local json = ngx.decode_base64(payload)
    if not json then
        return nil
    end

    local ok, claims = pcall(cjson.decode, json)
    if not ok or type(claims) ~= "table" then
        return nil
    end
    return claims
end

return _M
