-- auth_token_validator_test.lua
-- Runs the validator script against a fake nginx in which every shared dict
-- operation first yields, the way another worker may run between two of them.
describe("auth_token_validator", function()
    local script_path = "lua/auth/auth_token_validator.lua"
    local saved_ngx, saved_config, saved_utils
    local dict, headers_by_run

    -- Shared dict whose operations are each atomic, as in nginx, but may be
    -- interleaved with another request's operations
    local function interleaving_dict()
        local data = {}
        return {
            data = data,
            get = function(_, key)
                coroutine.yield()
                return data[key]
            end,
            set = function(_, key, value)
                coroutine.yield()
                data[key] = value
                return true
            end,
            incr = function(_, key, value, init)
                coroutine.yield()
                if data[key] == nil then
                    if init == nil then return nil, "not found" end
                    data[key] = init
                end
                data[key] = data[key] + value
                return data[key]
            end,
        }
    end

    local function fake_ngx(go_status)
        local current
        return {
            WARN = "warn",
            HTTP_GET = "GET",
            log = function() end,
            header = setmetatable({}, {
                __newindex = function(_, k, v) current.headers[k] = v end,
                __index = function(_, k) return current.headers[k] end,
            }),
            req = { set_uri_args = function() end },
            location = {
                capture = function()
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
    -- returns the status each one exited with
    local function run_concurrently(count)
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

    local function setup_fakes(limit, go_status)
        dict = interleaving_dict()
        _G.ngx = fake_ngx(go_status)
        package.loaded["auth.auth_config"] = {
            get_requests_per_token = function() return limit end,
            get_token_expiry_minutes = function() return 10 end,
            get_go_auth_service_url = function() return "http://auth" end,
        }
        package.loaded["utils.auth_utils"] = {
            extract_jwt_token = function() return "token-1", "header" end,
        }
    end

    setup(function()
        require("spec_helper")
    end)

    before_each(function()
        saved_ngx = _G.ngx
        saved_config = package.loaded["auth.auth_config"]
        saved_utils = package.loaded["utils.auth_utils"]
    end)

    after_each(function()
        _G.ngx = saved_ngx
        package.loaded["auth.auth_config"] = saved_config
        package.loaded["utils.auth_utils"] = saved_utils
    end)

    describe("with a token already validated", function()
        it("lets no more concurrent requests through than the limit", function()
            setup_fakes(3)
            dict.data["jwt_valid:token-1"] = "valid"

            local statuses = run_concurrently(8)

            assert.are.equal(3, count(statuses, 200))
            assert.are.equal(5, count(statuses, 401))
        end)

        it("reports how many requests remain", function()
            setup_fakes(3)
            dict.data["jwt_valid:token-1"] = "valid"

            run_concurrently(1)

            assert.are.equal("3", headers_by_run[1]["X-RateLimit-Limit"])
            assert.are.equal("2", headers_by_run[1]["X-RateLimit-Remaining"])
        end)

        it("reports none remaining once the limit is reached", function()
            setup_fakes(1)
            dict.data["jwt_valid:token-1"] = "valid"

            local statuses = run_concurrently(2)

            local denied = statuses[1] == 401 and 1 or 2
            assert.are.equal("0", headers_by_run[denied]["X-RateLimit-Remaining"])
        end)
    end)

    describe("with a token seen for the first time", function()
        it("counts every concurrent first request against the limit", function()
            setup_fakes(2)

            local statuses = run_concurrently(5)

            assert.are.equal(2, count(statuses, 200))
            assert.are.equal(3, count(statuses, 401))
        end)

        it("rejects the token when the auth service does", function()
            setup_fakes(5, 403)

            local statuses = run_concurrently(1)

            assert.are.same({ 401 }, statuses)
        end)
    end)
end)
