local json = require("cjson")
local M = {}

local function escape(value)
    return value:gsub("\\", "\\\\"):gsub('"', '\\"'):gsub("\n", "\\n")
end

-- Only configured provider metadata is used; URLs and credentials never become labels.
function M.record(provider)
    local name = provider.name or provider.type or "unknown"
    local chain_id = type(provider.chainId) == "number"
        and string.format("%.0f", provider.chainId) or tostring(provider.chainId or "unknown")
    local key = json.encode({ tostring(name), chain_id })
    local value, err = ngx.shared.upstream_metrics:incr(key, 1, 0)
    if not value then
        ngx.log(ngx.ERR, "Failed to record upstream request metric: ", err)
    end
end

function M.render()
    local lines = {
        "# HELP rpc_proxy_upstream_requests_total Outbound RPC request attempts to providers, including retries and failures",
        "# TYPE rpc_proxy_upstream_requests_total counter"
    }
    local dict = ngx.shared.upstream_metrics
    for _, key in ipairs(dict:get_keys(0)) do
        local value = dict:get(key)
        if value then
            local labels = json.decode(key)
            lines[#lines + 1] = string.format(
                'rpc_proxy_upstream_requests_total{provider_name="%s",chain_id="%s"} %d',
                escape(labels[1]), escape(labels[2]), value)
        end
    end
    return table.concat(lines, "\n") .. "\n"
end

return M
