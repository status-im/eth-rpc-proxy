describe("upstream request metrics", function()
    local json = require("cjson")
    local metrics = require("utils.upstream_metrics")
    local saved_ngx, saved_modules, counters, responses, calls, cached
    local providers = {
        { name = "Alchemy", type = "alchemy", chainId = 1, url = "https://alchemy" },
        { name = "Infura", type = "infura", chainId = 1, url = "https://infura" }
    }

    local function count(name, chain_id)
        return counters[json.encode({ name, tostring(chain_id) })] or 0
    end

    before_each(function()
        saved_ngx = ngx
        saved_modules = {}
        for _, name in ipairs({ "resty.http", "cache.cache", "utils.request_body" }) do
            saved_modules[name] = package.loaded[name]
        end
        counters, responses, calls, cached = {}, {}, {}, nil
        _G.ngx = {
            ERR = "ERR", WARN = "WARN", AF_INET = 2,
            var = { uri = "/ethereum/mainnet" }, header = {},
            log = function() end, say = function() end,
            req = {
                read_body = function() end,
                get_body_data = function() return '{}' end,
                get_method = function() return "POST" end
            },
            shared = {
                providers = { get = function() return json.encode(providers) end },
                upstream_metrics = {
                    incr = function(_, key, delta, initial)
                        counters[key] = (counters[key] or initial) + delta
                        return counters[key]
                    end,
                    get = function(_, key) return counters[key] end,
                    get_keys = function()
                        local keys = {}
                        for key in pairs(counters) do keys[#keys + 1] = key end
                        return keys
                    end
                }
            }
        }
        package.loaded["resty.http"] = { new = function()
            return { request_uri = function(_, url)
                calls[#calls + 1] = url
                local response = responses[#calls]
                return response, response == nil and "connection failed" or nil
            end }
        end }
        package.loaded["cache.cache"] = {
            check_cache = function() return { cached_response = cached } end,
            save_to_cache = function() end
        }
        package.loaded["utils.request_body"] = { decode = function(body) return body end }
    end)

    after_each(function()
        _G.ngx = saved_ngx
        for _, name in ipairs({ "resty.http", "cache.cache", "utils.request_body" }) do
            package.loaded[name] = saved_modules[name]
        end
    end)

    local function run()
        dofile("lua/request_handler.lua")
    end

    it("counts a successful outbound request only for the selected provider", function()
        responses[1] = { status = 200, body = '{"result":1}', headers = {} }
        run()
        assert.are.equal(1, count("Alchemy", 1))
        assert.are.equal(0, count("Infura", 1))
        assert.are.equal(1, #calls)
    end)

    it("does not count cache hits", function()
        cached = '{"result":1}'
        run()
        assert.are.equal(0, #calls)
        assert.is_nil(next(counters))
    end)

    it("counts both attempts when an HTTP error triggers failover", function()
        responses[1] = { status = 429, body = '{}', headers = {} }
        responses[2] = { status = 200, body = '{"result":1}', headers = {} }
        run()
        assert.are.equal(1, count("Alchemy", 1))
        assert.are.equal(1, count("Infura", 1))
        assert.are.equal(200, ngx.status)
    end)

    it("counts failed connections even when all providers fail", function()
        run()
        assert.are.equal(1, count("Alchemy", 1))
        assert.are.equal(1, count("Infura", 1))
        assert.are.equal(502, ngx.status)
    end)

    it("counts JSON-RPC failover attempts", function()
        responses[1] = { status = 200, body = '{"error":{"code":32005}}', headers = {} }
        responses[2] = { status = 200, body = '{"result":1}', headers = {} }
        run()
        assert.are.equal(1, count("Alchemy", 1))
        assert.are.equal(1, count("Infura", 1))
    end)

    it("does not count rejected routes", function()
        ngx.var.uri = "/invalid"
        run()
        assert.are.equal(400, ngx.status)
        assert.is_nil(next(counters))
    end)

    it("aggregates repeated attempts and separates chains", function()
        metrics.record(providers[1])
        metrics.record(providers[1])
        metrics.record({ name = "Alchemy", chainId = 10 })
        assert.are.equal(2, count("Alchemy", 1))
        assert.are.equal(1, count("Alchemy", 10))
    end)

    it("escapes labels and never exposes credentials or URLs", function()
        metrics.record({ name = 'Provider"\\\n', chainId = 1,
            url = "https://secret-url", authToken = "secret-token" })
        local output = metrics.render()
        assert.truthy(output:find('provider_name="Provider\\"\\\\\\n",chain_id="1"} 1', 1, true))
        assert.is_nil(output:find("secret", 1, true))
        assert.truthy(output:find("# TYPE rpc_proxy_upstream_requests_total counter", 1, true))
    end)

    it("uses bounded fallback labels for legacy providers", function()
        metrics.record({ type = "infura", url = "https://private" })
        metrics.record({ url = "https://another-private" })
        assert.are.equal(1, count("infura", "unknown"))
        assert.are.equal(1, count("unknown", "unknown"))
    end)

    it("exposes provider counters through the nginx metrics endpoint", function()
        metrics.record(providers[1])
        local output = {}
        ngx.print = function(...)
            local parts = { ... }
            for i, part in ipairs(parts) do parts[i] = tostring(part) end
            output[#output + 1] = table.concat(parts)
        end
        ngx.say = ngx.print
        ngx.shared.stats = { get = function() return 0 end }
        ngx.location = { capture = function()
            return { status = 200, body = "Active connections: 1\nserver accepts handled requests\n 2 2 3\nReading: 0 Writing: 1 Waiting: 0\n" }
        end }
        local file = assert(io.open("metrics.conf"))
        local source = file:read("*all")
        file:close()
        local block = assert(source:match("content_by_lua_block%s*{(.*)}%s*}"))
        assert(loadstring or load)(block)()
        local text = table.concat(output, "\n")
        assert.truthy(text:find('rpc_proxy_upstream_requests_total{provider_name="Alchemy",chain_id="1"} 1', 1, true))
        assert.truthy(text:find("nginx_http_requests_total 3", 1, true))
    end)

    it("does not break routing if the metrics dictionary is full", function()
        ngx.shared.upstream_metrics.incr = function() return nil, "no memory" end
        responses[1] = { status = 200, body = '{"result":1}', headers = {} }
        run()
        assert.are.equal(200, ngx.status)
        assert.are.equal(1, #calls)
    end)
end)
