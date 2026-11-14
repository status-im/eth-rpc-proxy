-- auth_token_validator_test.lua
-- Runs the validator script against a fake nginx in which every shared dict
-- operation first yields, the way another worker may run between two of them.
describe("auth_token_validator", function()
    local script_path = "lua/auth/auth_token_validator.lua"
    local NOW = 1000
    local saved_ngx, saved_config, saved_utils, saved_claims
    local dict, headers_by_run, go_calls, current_token, claims_by_token

    -- Shared dict whose operations are each atomic, as in nginx, but may be
    -- interleaved with another request's operations
    local function interleaving_dict()
        local data, ttls = {}, {}
        return {
            data = data,
            ttls = ttls,
            get = function(_, key)
                coroutine.yield()
                return data[key]
            end,
            set = function(_, key, value, ttl)
                coroutine.yield()
                data[key] = value
                ttls[key] = ttl
                return true
            end,
            incr = function(_, key, value, init, init_ttl)
                coroutine.yield()
                if data[key] == nil then
                    if init == nil then return nil, "not found" end
                    data[key] = init
                    ttls[key] = init_ttl
                end
                data[key] = data[key] + value
                return data[key]
            end,
            ttl = function(_, key)
                coroutine.yield()
                if data[key] == nil then return nil, "not found" end
                return ttls[key]
            end,
            expire = function(_, key, ttl)
                coroutine.yield()
                if data[key] == nil then return nil, "not found" end
                ttls[key] = ttl
                return true
            end,
        }
    end

    local function fake_ngx(go_status)
        local current
        return {
            WARN = "warn",
            ERR = "error",
            HTTP_GET = "GET",
            log = function() end,
            time = function() return NOW end,
            header = setmetatable({}, {
                __newindex = function(_, k, v) current.headers[k] = v end,
                __index = function(_, k) return current.headers[k] end,
            }),
            req = { set_uri_args = function() end },
            location = {
                capture = function()
                    go_calls = go_calls + 1
                    coroutine.yield()
                    return { status = go_status or 200 }
                end,
            },
            shared = { jwt_tokens = dict },
            -- Ends the run for good: it is never resumed. Raising an error
            -- instead would need a pcall around the script, which Lua 5.1
            -- cannot yield across.
            exit = function(status)
                current.status = status
                coroutine.yield()
            end,
            -- The run whose headers ngx.header writes to
            begin = function(run) current = run end,
        }
    end

    -- Runs the validator once per request, interleaved round-robin, and
    -- returns the status each one exited with. Every request carries token.
    local function run_concurrently(count, token)
        current_token = token or "token-1"
        local chunk = assert(loadfile(script_path))
        local runs = {}
        for i = 1, count do
            local run = { headers = {} }
            run.co = coroutine.create(chunk)
            runs[i] = run
        end
        local pending = count
        while pending > 0 do
            pending = 0
            for _, run in ipairs(runs) do
                if run.status == nil then
                    ngx.begin(run)
                    assert(coroutine.resume(run.co))
                    if run.status == nil and coroutine.status(run.co) == "dead" then
                        run.status = "fell through"
                    end
                    if run.status == nil then
                        pending = pending + 1
                    end
                end
            end
        end
        local statuses = {}
        headers_by_run = {}
        for i, run in ipairs(runs) do
            statuses[i] = run.status
            headers_by_run[i] = run.headers
        end
        return statuses
    end

    local function count(statuses, status)
        local n = 0
        for _, s in ipairs(statuses) do
            if s == status then n = n + 1 end
        end
        return n
    end

    -- limit: requests per token; go_status: what the auth service answers.
    -- Tokens carry the claims set in claims_by_token; token-1 and token-2 are
    -- two tokens of one solved puzzle.
    local function setup_fakes(limit, go_status)
        dict = interleaving_dict()
        go_calls = 0
        claims_by_token = {
            ["token-1"] = { jti = "puzzle-1", exp = NOW + 600 },
            ["token-2"] = { jti = "puzzle-1", exp = NOW + 600 },
        }
        _G.ngx = fake_ngx(go_status)
        package.loaded["auth.auth_config"] = {
            get_requests_per_token = function() return limit end,
            get_token_expiry_minutes = function() return 10 end,
        }
        package.loaded["utils.auth_utils"] = {
            extract_jwt_token = function() return current_token, "header" end,
        }
        package.loaded["auth.jwt_claims"] = {
            decode = function(token) return claims_by_token[token] end,
        }
    end

    setup(function()
        require("spec_helper")
    end)

    before_each(function()
        saved_ngx = _G.ngx
        saved_config = package.loaded["auth.auth_config"]
        saved_utils = package.loaded["utils.auth_utils"]
        saved_claims = package.loaded["auth.jwt_claims"]
    end)

    after_each(function()
        _G.ngx = saved_ngx
        package.loaded["auth.auth_config"] = saved_config
        package.loaded["utils.auth_utils"] = saved_utils
        package.loaded["auth.jwt_claims"] = saved_claims
    end)

    describe("with a token already validated", function()
        it("lets no more concurrent requests through than the limit", function()
            setup_fakes(3)
            dict.data["jwt_valid:token-1"] = "puzzle-1"

            local statuses = run_concurrently(8)

            assert.are.equal(3, count(statuses, 200))
            assert.are.equal(5, count(statuses, 401))
        end)

        it("reports how many requests remain", function()
            setup_fakes(3)
            dict.data["jwt_valid:token-1"] = "puzzle-1"

            run_concurrently(1)

            assert.are.equal("3", headers_by_run[1]["X-RateLimit-Limit"])
            assert.are.equal("2", headers_by_run[1]["X-RateLimit-Remaining"])
        end)

        it("reports none remaining once the limit is reached", function()
            setup_fakes(1)
            dict.data["jwt_valid:token-1"] = "puzzle-1"

            local statuses = run_concurrently(2)

            local denied = statuses[1] == 401 and 1 or 2
            assert.are.equal("0", headers_by_run[denied]["X-RateLimit-Remaining"])
        end)

        it("does not ask the auth service", function()
            setup_fakes(3)
            dict.data["jwt_valid:token-1"] = "puzzle-1"

            run_concurrently(2)

            assert.are.equal(0, go_calls)
        end)
    end)

    describe("with a token seen for the first time", function()
        it("counts every concurrent first request against the limit", function()
            setup_fakes(2)

            local statuses = run_concurrently(5)

            assert.are.equal(2, count(statuses, 200))
            assert.are.equal(3, count(statuses, 401))
        end)

        it("asks the auth service once the token is validated, not per request", function()
            setup_fakes(5)

            run_concurrently(1)
            run_concurrently(1)
            run_concurrently(1)

            assert.are.equal(1, go_calls)
        end)

        it("rejects the token when the auth service does", function()
            setup_fakes(5, 403)

            local statuses = run_concurrently(1)

            assert.are.same({ 401 }, statuses)
        end)

        it("does not remember a token the auth service rejected", function()
            setup_fakes(5, 403)

            run_concurrently(1)

            assert.is_nil(dict.data["jwt_valid:token-1"])
        end)

        it("rejects the token the auth service has seen too often", function()
            setup_fakes(5, 429)

            local statuses = run_concurrently(1)

            assert.are.same({ 401 }, statuses)
            assert.are.equal("0", headers_by_run[1]["X-RateLimit-Remaining"])
        end)
    end)

    describe("with several tokens of one solved puzzle", function()
        it("counts them against one limit", function()
            setup_fakes(3)

            local first = run_concurrently(2, "token-1")
            local second = run_concurrently(2, "token-2")

            assert.are.equal(2, count(first, 200))
            assert.are.equal(1, count(second, 200))
            assert.are.equal(1, count(second, 401))
        end)

        it("counts a token without an id on its own", function()
            setup_fakes(1)
            claims_by_token["token-2"] = { exp = NOW + 600 }

            local first = run_concurrently(1, "token-1")
            local second = run_concurrently(1, "token-2")

            assert.are.same({ 200 }, first)
            assert.are.same({ 200 }, second)
        end)

        it("counts a token whose claims cannot be read on its own", function()
            setup_fakes(1)
            claims_by_token["token-2"] = nil

            assert.are.same({ 200 }, run_concurrently(1, "token-2"))
            assert.are.same({ 401 }, run_concurrently(1, "token-2"))
        end)
    end)

    describe("how long a token is remembered", function()
        it("is until the token expires", function()
            setup_fakes(5)
            claims_by_token["token-1"].exp = NOW + 90

            run_concurrently(1)

            assert.are.equal(90, dict.ttls["jwt_valid:token-1"])
        end)

        it("is never longer than the configured token lifetime", function()
            setup_fakes(5)
            claims_by_token["token-1"].exp = NOW + 100000

            run_concurrently(1)

            assert.are.equal(600, dict.ttls["jwt_valid:token-1"])
        end)

        it("is the configured token lifetime when the token names no expiry", function()
            setup_fakes(5)
            claims_by_token["token-1"].exp = nil

            run_concurrently(1)

            assert.are.equal(600, dict.ttls["jwt_valid:token-1"])
        end)

        it("keeps the usage counter longer than the token", function()
            setup_fakes(5)
            claims_by_token["token-1"].exp = NOW + 90

            run_concurrently(1)

            assert.are.equal(150, dict.ttls["jwt_usage:puzzle-1"])
        end)

        it("keeps the usage counter for a later token of the same puzzle", function()
            setup_fakes(5)
            claims_by_token["token-1"].exp = NOW + 90
            claims_by_token["token-2"].exp = NOW + 500

            run_concurrently(1, "token-1")
            run_concurrently(1, "token-2")

            assert.are.equal(560, dict.ttls["jwt_usage:puzzle-1"])
        end)

        it("does not cut the usage counter short for an earlier token", function()
            setup_fakes(5)
            claims_by_token["token-1"].exp = NOW + 500
            claims_by_token["token-2"].exp = NOW + 90

            run_concurrently(1, "token-1")
            run_concurrently(1, "token-2")

            assert.are.equal(560, dict.ttls["jwt_usage:puzzle-1"])
        end)
    end)

    describe("when the auth service fails", function()
        it("answers with a server error, not a rejection of the token", function()
            setup_fakes(5, 502)

            local statuses = run_concurrently(1)

            assert.are.same({ 503 }, statuses)
        end)

        it("does not remember the token", function()
            setup_fakes(5, 504)

            run_concurrently(1)

            assert.is_nil(dict.data["jwt_valid:token-1"])
        end)

        it("does not count the request", function()
            setup_fakes(5, 502)

            run_concurrently(1)

            assert.is_nil(dict.data["jwt_usage:puzzle-1"])
        end)
    end)

    describe("without a token", function()
        it("rejects the request without asking the auth service", function()
            setup_fakes(5)
            package.loaded["utils.auth_utils"] = {
                extract_jwt_token = function() return nil end,
            }

            local statuses = run_concurrently(1)

            assert.are.same({ 401 }, statuses)
            assert.are.equal(0, go_calls)
        end)
    end)
end)
