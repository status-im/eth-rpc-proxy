local provider_loader = require("provider_loader")
local schedule_reload_providers = provider_loader.schedule_reload_providers

-- Read URL from environment variable
local url = os.getenv("CONFIG_HEALTH_CHECKER_URL")
local fallback = "/usr/local/openresty/nginx/providers.json"

-- Check worker ID to ensure timers only run in one process
if ngx.worker.id() == 0 then  -- Only in first worker
    ngx.log(ngx.INFO, "Starting reload_providers in worker: ", ngx.worker.id())

    -- Perform initial provider loading
    schedule_reload_providers(url, fallback)

    -- Start periodic reload
    local delay = tonumber(os.getenv("RELOAD_INTERVAL")) or 30
    local handler
    handler = function()
        schedule_reload_providers(url, fallback)
        local ok, err = ngx.timer.at(delay, handler)
        if not ok then
            ngx.log(ngx.ERR, "Failed to create timer: ", err)
        end
    end

    local ok, err = ngx.timer.at(delay, handler)
    if not ok then
        ngx.log(ngx.ERR, "Failed to create initial timer: ", err)
    end
else
    ngx.log(ngx.ERR, "Worker ", ngx.worker.id(), " is not starting reload_providers")
end 