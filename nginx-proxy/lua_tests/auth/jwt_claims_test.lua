-- jwt_claims_test.lua
describe("jwt_claims", function()
    local jwt_claims
    local saved_ngx

    local ALPHABET = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/"

    -- Standard base64 with padding, as ngx.decode_base64 takes it; nil for
    -- anything else
    local function decode_base64(input)
        if #input % 4 ~= 0 or input:find("[^%w%+/=]") then
            return nil
        end
        local out = {}
        for quad in input:gmatch("....") do
            local n, pad = 0, 0
            for i = 1, 4 do
                local c = quad:sub(i, i)
                if c == "=" then
                    pad = pad + 1
                    n = n * 64
                else
                    n = n * 64 + ALPHABET:find(c, 1, true) - 1
                end
            end
            local bytes = string.char(math.floor(n / 65536) % 256, math.floor(n / 256) % 256, n % 256)
            out[#out + 1] = bytes:sub(1, 3 - pad)
        end
        return table.concat(out)
    end

    local function encode_base64url(input)
        local out = {}
        for i = 1, #input, 3 do
            local a, b, c = input:byte(i, i + 2)
            local n = a * 65536 + (b or 0) * 256 + (c or 0)
            local chars = {}
            for shift = 3, 0, -1 do
                local index = math.floor(n / 64 ^ shift) % 64
                chars[#chars + 1] = ALPHABET:sub(index + 1, index + 1)
            end
            local keep = c and 4 or (b and 3 or 2)
            out[#out + 1] = table.concat(chars, "", 1, keep)
        end
        return (table.concat(out):gsub("%+", "-"):gsub("/", "_"))
    end

    local function token_with_payload(json)
        return "eyJhbGciOiJIUzI1NiJ9." .. encode_base64url(json) .. ".c2lnbmF0dXJl"
    end

    setup(function()
        require("spec_helper")
        jwt_claims = require("auth.jwt_claims")
    end)

    before_each(function()
        saved_ngx = _G.ngx
        _G.ngx = { decode_base64 = decode_base64 }
    end)

    after_each(function()
        _G.ngx = saved_ngx
    end)

    it("reads the claims of a token", function()
        local claims = jwt_claims.decode(token_with_payload('{"jti":"puzzle-1","exp":1700000600}'))

        assert.are.equal("puzzle-1", claims.jti)
        assert.are.equal(1700000600, claims.exp)
    end)

    it("reads a payload of any length", function()
        for _, jti in ipairs({ "a", "ab", "abc", "abcd" }) do
            local claims = jwt_claims.decode(token_with_payload('{"jti":"' .. jti .. '"}'))

            assert.are.equal(jti, claims.jti)
        end
    end)

    it("reads a payload that uses the url-safe alphabet", function()
        -- "??>>" encodes with both characters that differ from plain base64
        local token = token_with_payload('{"jti":"??>>"}')
        assert.is_truthy(token:find("[-_]"))

        assert.are.equal("??>>", jwt_claims.decode(token).jti)
    end)

    it("returns nothing for a token that is not three parts", function()
        assert.is_nil(jwt_claims.decode("not-a-token"))
        assert.is_nil(jwt_claims.decode("only.two"))
        assert.is_nil(jwt_claims.decode("one.two.three.four"))
    end)

    it("returns nothing for a payload that is not base64", function()
        assert.is_nil(jwt_claims.decode("header.!!!!.signature"))
        assert.is_nil(jwt_claims.decode("header.abcde.signature"))
    end)

    it("returns nothing for a payload that is not a JSON object", function()
        assert.is_nil(jwt_claims.decode(token_with_payload("not json")))
        assert.is_nil(jwt_claims.decode(token_with_payload("42")))
    end)
end)
