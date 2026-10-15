describe("client RPC request metrics", function()
    local json = require("cjson")
    local metrics = require("utils.client_metrics")
    local saved_ngx, saved_modules, counters, routes, responses, calls, cached, body
    local providers = {
        { name = "Alchemy", type = "alchemy", chainId = 1, url = "https://alchemy" },
        { name = "Infura", type = "infura", chainId = 1, url = "https://infura" }
    }

    local function dict(storage)
        return {
            safe_add = function(_, key, value)
                if storage[key] ~= nil then return nil, "exists" end
                storage[key] = value
                return true
            end,
            safe_set = function(_, key, value) storage[key] = value; return true end,
            incr = function(_, key, delta)
                if storage[key] == nil then return nil, "not found" end
                storage[key] = storage[key] + delta
                return storage[key]
            end,
            get = function(_, key) return storage[key] end,
            get_keys = function()
                local keys = {}
                for key in pairs(storage) do keys[#keys + 1] = key end
                return keys
            end
        }
    end

    local function count(chain, network, provider_type, method)
        return counters[json.encode({ chain, network, provider_type, method })] or 0
    end

    local function total()
        local sum = 0
        for _, value in pairs(counters) do sum = sum + value end
        return sum
    end

    before_each(function()
        saved_ngx = ngx
        saved_modules = {}
        for _, name in ipairs({ "resty.http", "cache.cache", "utils.request_body" }) do
            saved_modules[name] = package.loaded[name]
        end
        counters, routes, responses, calls, cached = {}, {}, {}, {}, nil
        body = '{"jsonrpc":"2.0","method":"eth_chainId","id":1}'
        _G.ngx = {
            ERR = "ERR", WARN = "WARN", AF_INET = 2,
            ctx = {}, var = { uri = "/ethereum/mainnet" }, header = {},
            log = function() end, say = function() end,
            req = {
                read_body = function() end,
                get_body_data = function() return body end,
                get_method = function() return "POST" end
            },
            shared = {
                providers = { get = function() return json.encode(providers) end },
                client_metrics = dict(counters), client_routes = dict(routes),
                stats = { incr = function() end }
            }
        }
        assert(metrics.configure(json.encode({ chains = {
            { name = "ethereum", network = "mainnet", providers = providers }
        } })))
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
        package.loaded["utils.request_body"] = { decode = function(data) return data end }
    end)

    after_each(function()
        _G.ngx = saved_ngx
        for _, name in ipairs({ "resty.http", "cache.cache", "utils.request_body" }) do
            package.loaded[name] = saved_modules[name]
        end
    end)

    local function run()
        dofile("lua/request_handler.lua")
        metrics.finish()
    end

    it("counts a successful request once using the requested type, not the serving provider", function()
        responses[1] = { status = 200, body = '{"result":1}', headers = {} }
        run()
        assert.are.equal(1, count("ethereum", "mainnet", "auto", "eth_chainId"))
        assert.are.equal(1, total())
    end)

    it("counts a cache hit without contacting a provider", function()
        cached = '{"result":1}'
        run()
        assert.are.equal(0, #calls)
        assert.are.equal(1, total())
    end)

    it("counts HTTP failover once", function()
        responses[1] = { status = 429, body = '{}', headers = {} }
        responses[2] = { status = 200, body = '{"result":1}', headers = {} }
        run()
        assert.are.equal(2, #calls)
        assert.are.equal(1, total())
    end)

    it("counts one client request when all connections fail", function()
        run()
        assert.are.equal(2, #calls)
        assert.are.equal(1, total())
        assert.are.equal(502, ngx.status)
    end)

    it("counts JSON-RPC failover once", function()
        responses[1] = { status = 200, body = '{"error":{"code":32005}}', headers = {} }
        responses[2] = { status = 200, body = '{"result":1}', headers = {} }
        run()
        assert.are.equal(2, #calls)
        assert.are.equal(1, total())
    end)

    it("records an explicit requested provider type even on a cache hit", function()
        ngx.var.uri = "/ethereum/mainnet/infura"
        cached = '{"result":1}'
        run()
        assert.are.equal(1, count("ethereum", "mainnet", "infura", "eth_chainId"))
    end)

    it("collapses unknown routes and types into one bounded category", function()
        for _, uri in ipairs({ "/invalid", "/foo/bar", "/ethereum/mainnet/arbitrary" }) do
            ngx.ctx = {}; ngx.var.uri = uri
            run()
        end
        assert.are.equal(3, count("unknown", "unknown", "unknown", "eth_chainId"))
        assert.are.equal(3, total())
        assert.are.equal(100, #ngx.shared.client_metrics:get_keys(0))
    end)

    it("keeps configured labels when health removes every provider", function()
        ngx.shared.providers.get = function() return nil end
        ngx.var.uri = "/ethereum/mainnet/infura"
        run()
        assert.are.equal(404, ngx.status)
        assert.are.equal(1, count("ethereum", "mainnet", "infura", "eth_chainId"))
    end)

    it("counts a batch once", function()
        body = '[{"method":"eth_call"},{"method":"eth_chainId"}]'
        cached = '{}'
        run()
        assert.are.equal(1, count("ethereum", "mainnet", "auto", "batch"))
        assert.are.equal(1, total())
    end)

    it("classifies array-shaped bodies as batch without validating their contents", function()
        for _, batch in ipairs({ '[]', '[1]', ' \n\t[broken', '[{"method":"eth_call"}]' }) do
            assert.are.equal("batch", metrics.classify(batch))
        end
        -- The metric must not parse large batch contents just to categorize them.
        assert.are.equal("batch", metrics.classify("[" .. string.rep("x", 10 * 1024 * 1024 - 1)))
    end)

    it("bounds unknown method names", function()
        for i = 1, 100 do
            body = '{"method":"random_' .. i .. '"}'
            ngx.ctx = {}; cached = '{}'
            run()
        end
        assert.are.equal(100, count("ethereum", "mainnet", "auto", "other"))
        assert.are.equal(100, #ngx.shared.client_metrics:get_keys(0))
    end)

    it("classifies malformed bodies and decoding failures as invalid", function()
        for _, invalid in ipairs({ 'broken', '{}', 'null', '{"method":2}' }) do
            assert.are.equal("invalid", metrics.classify(invalid))
        end
        package.loaded["utils.request_body"].decode = function() return nil, 400, "bad gzip" end
        run()
        assert.are.equal(400, ngx.status)
        assert.are.equal(1, count("ethereum", "mainnet", "auto", "invalid"))
    end)

    it("counts a body rejected after authorization in the log phase", function()
        ngx.req.read_body = function() error("nginx stopped reading oversized body") end
        assert.has_error(function() dofile("lua/request_handler.lua") end)
        metrics.finish()
        assert.are.equal(1, count("ethereum", "mainnet", "auto", "invalid"))
    end)

    it("does not count auth rejection, service endpoints or subrequests", function()
        for _, uri in ipairs({ "/ethereum/mainnet", "/auth/verify", "/metrics", "/_auth_token" }) do
            ngx.ctx = {}; ngx.var.uri = uri
            metrics.finish()
        end
        assert.are.equal(0, total())
    end)

    it("does not count a completed request twice", function()
        cached = '{}'
        run()
        metrics.finish()
        assert.are.equal(1, total())
    end)

    it("preinitializes zeros and preserves counters on configuration refresh", function()
        assert.are.equal(100, #ngx.shared.client_metrics:get_keys(0))
        metrics.record("/ethereum/mainnet", "eth_chainId")
        assert(metrics.configure(json.encode({ chains = {
            { name = "ethereum", network = "mainnet", providers = providers }
        } })))
        assert.are.equal(1, total())
    end)

    it("counts requests served before worker zero initializes metric routes and counters", function()
        routes.routes = nil
        counters = {}
        ngx.shared.client_metrics = dict(counters)
        metrics.record("/ethereum/mainnet", "eth_chainId")
        assert.are.equal(1, count("unknown", "unknown", "unknown", "eth_chainId"))
    end)

    it("preserves both requests when another worker creates a missing counter first", function()
        local key = json.encode({ "ethereum", "mainnet", "auto", "eth_chainId" })
        counters[key] = nil
        ngx.shared.client_metrics.safe_add = function(_, added_key)
            -- Another worker initialized and incremented between our incr and safe_add.
            counters[added_key] = 1
            return nil, "exists"
        end
        metrics.record("/ethereum/mainnet", "eth_chainId")
        assert.are.equal(2, count("ethereum", "mainnet", "auto", "eth_chainId"))
    end)

    it("logs allocation failure without dropping existing counters or failing the request", function()
        local key = json.encode({ "ethereum", "mainnet", "auto", "eth_chainId" })
        counters[key] = nil
        metrics.record("/ethereum/mainnet", "net_version")
        local logged_error
        ngx.log = function(_, _, err) logged_error = err end
        ngx.shared.client_metrics.safe_add = function() return nil, "no memory" end
        cached = '{}'
        run()
        assert.are.equal("no memory", logged_error)
        assert.are.equal(1, count("ethereum", "mainnet", "auto", "net_version"))
        assert.are.equal(0, count("ethereum", "mainnet", "auto", "eth_chainId"))
        assert.are.equal(0, #calls)
    end)

    it("does not evict existing counters or replace routes on allocation failure", function()
        local previous = routes.routes
        ngx.shared.client_metrics.safe_add = function() return nil, "no memory" end
        assert.is_nil(metrics.configure('{"chains":[]}'))
        assert.are.equal(previous, routes.routes)
    end)

    it("refreshes configured route labels without losing prior counters", function()
        metrics.record("/base/mainnet", "eth_chainId")
        assert.are.equal(1, count("unknown", "unknown", "unknown", "eth_chainId"))
        assert(metrics.configure(json.encode({ chains = {
            { name = "ethereum", network = "mainnet", providers = providers },
            { name = "base", network = "mainnet", providers = {} }
        } })))
        metrics.record("/base/mainnet", "eth_chainId")
        assert.are.equal(1, count("base", "mainnet", "auto", "eth_chainId"))
        assert.are.equal(1, count("unknown", "unknown", "unknown", "eth_chainId"))
    end)

    it("keeps the previous route allowlist on malformed configuration", function()
        local previous = routes.routes
        for _, config in ipairs({ 'broken', '{"chains":{ "foo":1 }}', '{"chains":[1]}' }) do
            assert.is_nil(metrics.configure(config))
            assert.are.equal(previous, routes.routes)
        end
    end)

    it("does not break processing if metrics recording fails", function()
        ngx.shared.client_metrics.incr = function() return nil, "not found" end
        cached = '{}'
        run()
        assert.are.equal(0, total())
    end)

    it("exports client counters through the nginx metrics endpoint", function()
        metrics.record("/ethereum/mainnet", "eth_chainId")
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
        local source = file:read("*all"); file:close()
        local block = assert(source:match("content_by_lua_block%s*{(.*)}%s*}"))
        assert(loadstring or load)(block)()
        local text = table.concat(output, "\n")
        assert.truthy(text:find('rpc_proxy_client_requests_total{chain="ethereum",network="mainnet",requested_provider_type="auto",rpc_method="eth_chainId"} 1', 1, true))
        assert.is_nil(text:find("Alchemy", 1, true))
        assert.is_nil(text:find("https://", 1, true))
    end)
end)
