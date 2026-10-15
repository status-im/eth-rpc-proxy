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

local counters = {
    { name = "rpc_proxy_client_requests_total", prefix = "",
        help = "Authorized client RPC HTTP requests after body read, including cache hits and processing failures" },
    { name = "rpc_proxy_client_request_bytes_total", prefix = "request_bytes:",
        help = "Incoming client RPC HTTP bytes including request line, headers and encoded body" },
    { name = "rpc_proxy_client_response_bytes_total", prefix = "response_bytes:",
        help = "Outgoing client RPC HTTP bytes including response headers and encoded body" }
}

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
            for _, counter in ipairs(counters) do
                local added, err = ngx.shared.client_metrics:safe_add(counter.prefix .. key, 0)
                if not added and err ~= "exists" then return nil, err end
            end
        end
    end
    return ngx.shared.client_routes:safe_set("routes", json.encode(routes))
end

local function increment(counter, labels_key, amount)
    local key = counter.prefix .. labels_key
    local dict = ngx.shared.client_metrics
    local value, err = dict:incr(key, amount)
    if not value and err == "not found" then
        -- Workers can accept requests before worker 0 finishes initialization.
        local added, add_err = dict:safe_add(key, 0)
        if added or add_err == "exists" then
            value, err = dict:incr(key, amount)
        else
            err = add_err
        end
    end
    if not value then
        ngx.log(ngx.ERR, "Failed to record ", counter.name, ": ", err)
    end
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
    increment(counters[1], key, 1)
    increment(counters[2], key, tonumber(ngx.var.request_length) or 0)
    increment(counters[3], key, tonumber(ngx.var.bytes_sent) or 0)
end

-- Called by the RPC handler after authorization and successful HTTP body reading.
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
    local lines = {}
    local samples = {}
    for i in ipairs(counters) do samples[i] = {} end
    local dict = ngx.shared.client_metrics
    for _, key in ipairs(dict:get_keys(0)) do
        -- Unprefixed request counters define the label sets. This keeps their
        -- existing shared-memory keys and values intact across nginx reloads.
        if key:sub(1, 1) == "[" then
            local labels = json.decode(key)
            local label_text = string.format(
                '{chain="%s",network="%s",requested_provider_type="%s",rpc_method="%s"}',
                escape(labels[1]), escape(labels[2]), escape(labels[3]), escape(labels[4]))
            for i, counter in ipairs(counters) do
                local value = dict:get(counter.prefix .. key)
                if value then
                    samples[i][#samples[i] + 1] = counter.name .. label_text .. " " .. string.format("%d", value)
                end
            end
        end
    end
    for i, counter in ipairs(counters) do
        lines[#lines + 1] = "# HELP " .. counter.name .. " " .. counter.help
        lines[#lines + 1] = "# TYPE " .. counter.name .. " counter"
        for _, line in ipairs(samples[i]) do lines[#lines + 1] = line end
    end
    return table.concat(lines, "\n") .. "\n"
end

return M
