local json = require("cjson")
local request_utils = require("utils.request_utils")
local M = {}
local cached_routes_text
local cached_routes = {}

local methods = {
    "eth_getBlockByHash", "eth_getBlockByNumber", "eth_getTransactionByHash",
    "eth_getTransactionReceipt", "eth_getLogs", "eth_blockNumber", "eth_gasPrice",
    "eth_getBalance", "eth_getCode", "eth_getStorageAt", "eth_getTransactionCount",
    "eth_call", "eth_estimateGas", "eth_sendRawTransaction", "eth_sendTransaction",
    "web3_clientVersion", "web3_sha3", "net_version", "net_listening", "net_peerCount",
    "eth_chainId", "eth_syncing"
}
local allowed_methods = {}
for _, method in ipairs(methods) do allowed_methods[method] = true end
local categories = {}
for _, method in ipairs(methods) do categories[#categories + 1] = method end
for _, method in ipairs({ "other", "batch", "invalid" }) do categories[#categories + 1] = method end

local function escape(value)
    return value:gsub("\\", "\\\\"):gsub('"', '\\"'):gsub("\n", "\\n")
end

local function classify(body)
    if type(body) ~= "string" then return "invalid" end
    -- This is a metric category, not JSON validation. Avoid parsing batch payloads.
    if body:match("^%s*%[") then return "batch" end
    local ok, decoded = pcall(json.decode, body)
    if not ok or type(decoded) ~= "table" then return "invalid" end
    if type(decoded.method) ~= "string" or decoded.method == "" then return "invalid" end
    return allowed_methods[decoded.method] and decoded.method or "other"
end

-- Accept a validated route index from provider_config. Missing configuration
-- retains the previous allowlist, or initializes only unknown counters at startup.
-- Preinitialize counters before publishing routes so scrapes can observe zero.
function M.configure(routes)
    if not routes then
        if ngx.shared.client_routes:get("routes") then return true end
        routes = {}
    end
    local combinations = { { "unknown", "unknown", "unknown" } }
    for _, route in pairs(routes) do
        for provider_type in pairs(route.types) do
            combinations[#combinations + 1] = { route.chain, route.network, provider_type }
        end
    end
    for _, route in ipairs(combinations) do
        for _, method in ipairs(categories) do
            local key = json.encode({ route[1], route[2], route[3], method })
            local added, err = ngx.shared.client_metrics:safe_add(key, 0)
            if not added and err ~= "exists" then return nil, err end
        end
    end
    return ngx.shared.client_routes:safe_set("routes", json.encode(routes))
end

local function record(uri, method)
    local chain, network, provider_type = request_utils.parse_url_path(uri)
    local stored = ngx.shared.client_routes:get("routes")
    if stored ~= cached_routes_text then
        cached_routes = stored and json.decode(stored) or {}
        cached_routes_text = stored
    end
    local route = chain and cached_routes[chain .. ":" .. network]
    provider_type = provider_type or "auto"
    if not route or not route.types[provider_type] then
        chain, network, provider_type = "unknown", "unknown", "unknown"
    end
    method = allowed_methods[method] and method
        or (method == "other" or method == "batch") and method or "invalid"
    local key = json.encode({ chain, network, provider_type, method })
    local dict = ngx.shared.client_metrics
    local value, err = dict:incr(key, 1)
    if not value and err == "not found" then
        -- Workers can accept requests before worker 0 finishes initialization.
        local added, add_err = dict:safe_add(key, 0)
        if added or add_err == "exists" then
            value, err = dict:incr(key, 1)
        else
            err = add_err
        end
    end
    if not value then
        ngx.log(ngx.ERR, "Failed to record client RPC request metric: ", err)
    end
end

-- Called by the RPC content handler after nginx authorization and before body I/O.
function M.begin_request()
    ngx.ctx.client_rpc_admitted = true
    ngx.ctx.client_rpc_method = "invalid"
end

function M.observe_body(body)
    if ngx.ctx.client_rpc_admitted then
        ngx.ctx.client_rpc_method = classify(body)
    end
end

function M.finish()
    -- Internal auth subrequests and /metrics never begin a client RPC request.
    if ngx.ctx.client_rpc_admitted then
        local method = ngx.ctx.client_rpc_method
        ngx.ctx.client_rpc_admitted = nil
        ngx.ctx.client_rpc_method = nil
        record(ngx.var.uri, method)
    end
end

function M.render()
    local lines = {
        "# HELP rpc_proxy_client_requests_total Authorized client RPC HTTP requests, including cache hits and processing failures",
        "# TYPE rpc_proxy_client_requests_total counter"
    }
    local dict = ngx.shared.client_metrics
    for _, key in ipairs(dict:get_keys(0)) do
        local value = dict:get(key)
        if value then
            local labels = json.decode(key)
            lines[#lines + 1] = string.format(
                'rpc_proxy_client_requests_total{chain="%s",network="%s",requested_provider_type="%s",rpc_method="%s"} %d',
                escape(labels[1]), escape(labels[2]), escape(labels[3]), escape(labels[4]), value)
        end
    end
    return table.concat(lines, "\n") .. "\n"
end

return M
