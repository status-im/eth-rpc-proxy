local zlib = require("zlib")

local _M = {}

-- Request encodings the proxy decodes, as advertised in Accept-Encoding (RFC 7694)
_M.ACCEPTED_ENCODINGS = "gzip"

-- A decoded body may be as large as a plain one (client_max_body_size)
_M.MAX_DECODED_SIZE = 10 * 1024 * 1024

-- 15 window bits plus 16 selects the gzip wrapper only
local GZIP_WINDOW_BITS = 31

-- Deflate expands at most ~1032:1, so a chunk this size can overshoot the
-- limit by about a megabyte before it is caught
local INPUT_CHUNK = 1024

local function inflate_gzip(body)
    local stream = zlib.inflate(GZIP_WINDOW_BITS)
    local parts, size = {}, 0
    local eof, consumed = false, 0

    for offset = 1, #body, INPUT_CHUNK do
        if eof then
            return nil, 400, "Invalid gzip body: data after the gzip stream"
        end
        local ok, out, done, bytes_in = pcall(stream, body:sub(offset, offset + INPUT_CHUNK - 1))
        if not ok then
            return nil, 400, "Invalid gzip body"
        end
        size = size + #out
        if size > _M.MAX_DECODED_SIZE then
            return nil, 413, "Decoded request body is too large"
        end
        parts[#parts + 1] = out
        eof, consumed = done, bytes_in
    end

    if not eof then
        return nil, 400, "Invalid gzip body: truncated"
    end
    if consumed < #body then
        return nil, 400, "Invalid gzip body: data after the gzip stream"
    end
    return table.concat(parts)
end

-- Returns the body as the client meant it, or nil, an HTTP status and a
-- message when the Content-Encoding is unsupported or the body does not decode
function _M.decode(body, content_encoding)
    local encoding = (content_encoding or ""):lower():match("^%s*(.-)%s*$")
    if encoding == "" or encoding == "identity" then
        return body
    end
    if encoding == "gzip" or encoding == "x-gzip" then
        return inflate_gzip(body)
    end
    return nil, 415, "Unsupported Content-Encoding: " .. encoding
end

return _M
