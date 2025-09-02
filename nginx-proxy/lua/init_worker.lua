local provider_loader = require("providers.provider_loader")
local schedule_reload_providers = provider_loader.schedule_reload_providers

-- Initialize auth configuration
local auth_config = require("auth.auth_config")
auth_config.init()

-- Initialize cache diagnostics
local cache = require("cache.cache")

-- Run cache diagnostics on startup (only in first worker)
if ngx.worker.id() == 0 then
    pcall(cache.run_diagnostics)
end

-- Read URL from environment variable
local url = os.getenv("CONFIG_HEALTH_CHECKER_URL")
local fallback = "/app/providers.json"

-- Check worker ID to ensure timers only run in one process
if ngx.worker.id() == 0 then  -- Only in first worker
    ngx.log(ngx.INFO, "Starting reload_providers in worker: ", ngx.worker.id())

    -- Perform initial provider loading
    schedule_reload_providers(url, fallback)

    -- Start periodic reload
    local delay = tonumber(os.getenv("RELOAD_INTERVAL")) or 30
    local handler
    handler = function()
        local ok, err = pcall(schedule_reload_providers, url, fallback)
        if not ok then
            ngx.log(ngx.ERR, "Failed to execute schedule_reload_providers: ", err)
        end

        -- Reschedule timer regardless of the result
        local ok_timer, err_timer = ngx.timer.at(delay, handler)
        if not ok_timer then
            ngx.log(ngx.ERR, "Failed to reschedule timer: ", err_timer)
        end
    end

    local ok, err = ngx.timer.at(delay, handler)
    if not ok then
        ngx.log(ngx.ERR, "Failed to create initial timer: ", err)
    end
else
    ngx.log(ngx.ERR, "Worker ", ngx.worker.id(), " is not starting reload_providers")
end 