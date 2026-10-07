-- request_body_test.lua
describe("request_body", function()
    local request_body
    local zlib = require("zlib")

    local function gzip(data)
        return zlib.deflate(6, 31)(data, "finish")
    end

    local function zlib_wrapped(data)
        return zlib.deflate(6, 15)(data, "finish")
    end

    local call = '{"jsonrpc":"2.0","id":1,"method":"eth_call","params":[{"to":"0xca11","data":"0x82ad56cb' ..
        string.rep("0", 4096) .. '"},"latest"]}'

    setup(function()
        require("spec_helper")
        request_body = require("utils.request_body")
    end)

    describe("decode", function()
        it("passes a body without Content-Encoding through", function()
            assert.are.equal(call, request_body.decode(call, nil))
            assert.are.equal(call, request_body.decode(call, ""))
        end)

        it("passes an identity body through", function()
            assert.are.equal(call, request_body.decode(call, "identity"))
        end)

        it("inflates a gzip body", function()
            local compressed = gzip(call)
            assert.is_true(#compressed < #call / 10)
            assert.are.equal(call, request_body.decode(compressed, "gzip"))
        end)

        it("reads the encoding case-insensitively and ignores spaces", function()
            assert.are.equal(call, request_body.decode(gzip(call), " GZip "))
            assert.are.equal(call, request_body.decode(gzip(call), "x-gzip"))
        end)

        it("inflates an empty gzip body to an empty body", function()
            assert.are.equal("", request_body.decode(gzip(""), "gzip"))
        end)

        it("rejects an encoding it cannot decode with 415", function()
            for _, encoding in ipairs({ "br", "deflate", "zstd", "gzip, gzip" }) do
                local body, status = request_body.decode(call, encoding)
                assert.is_nil(body, encoding)
                assert.are.equal(415, status, encoding)
            end
        end)

        it("rejects a body that is not gzip with 400", function()
            local body, status = request_body.decode(call, "gzip")
            assert.is_nil(body)
            assert.are.equal(400, status)
        end)

        it("rejects zlib-wrapped deflate sent as gzip with 400", function()
            local body, status = request_body.decode(zlib_wrapped(call), "gzip")
            assert.is_nil(body)
            assert.are.equal(400, status)
        end)

        it("rejects a truncated gzip body with 400", function()
            local compressed = gzip(call)
            local body, status = request_body.decode(compressed:sub(1, #compressed - 8), "gzip")
            assert.is_nil(body)
            assert.are.equal(400, status)
        end)

        it("rejects data after the gzip stream with 400", function()
            local body, status = request_body.decode(gzip(call) .. "garbage", "gzip")
            assert.is_nil(body)
            assert.are.equal(400, status)
        end)

        it("rejects a body that inflates beyond the limit with 413", function()
            local bomb = gzip(string.rep("0", request_body.MAX_DECODED_SIZE + 1))
            assert.is_true(#bomb < 64 * 1024)
            local body, status = request_body.decode(bomb, "gzip")
            assert.is_nil(body)
            assert.are.equal(413, status)
        end)

        it("accepts a body that inflates exactly to the limit", function()
            local data = string.rep("0", request_body.MAX_DECODED_SIZE)
            local body = request_body.decode(gzip(data), "gzip")
            assert.are.equal(#data, #body)
        end)
    end)
end)
