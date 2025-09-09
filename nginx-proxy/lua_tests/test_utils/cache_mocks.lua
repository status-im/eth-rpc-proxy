-- Simplified cache_mocks.lua for go-proxy-cache integration
-- Only mocks the HTTP communication, not internal cache logic

local _M = {}

-- Simple HTTP mock for go-proxy-cache communication
function _M.setup_go_cache_http_mock(response_data)
  response_data = response_data or {
    success = true,
    found = false,
    cache_status = "MISS",
    cache_type = "short"
  }
  
  package.preload["resty.http"] = function()
    return {
      new = function()
        return {
          set_timeout = function() end,
          connect = function() return true, nil end,
          request = function()
            return {
              status = 200,
              read_body = function()
                return require("cjson").encode(response_data)
              end
            }, nil
          end,
          set_keepalive = function() end
        }
      end
    }
  end
end

-- Mock environment variables
function _M.setup_cache_env_vars()
  _G.os = _G.os or {}
  _G.os.getenv = function(var)
    if var == "GO_CACHE_SOCKET" then
      return "/tmp/cache.sock"
    end
    return nil
  end
end

-- Mock cache diagnostics
function _M.setup_cache_diagnostics_mock()
  package.preload["utils.cache_diagnostics"] = function()
    return {
      test_cache_connectivity = function()
        return true
      end
    }
  end
end

-- Legacy shared dicts (minimal, for compatibility)
function _M.setup_cache_shared_dicts()
  if not _G.ngx then
    error("nginx mocks must be setup first")
  end
  
  _G.ngx.shared = _G.ngx.shared or {}
  -- Only create empty mocks, not used anymore
  _G.ngx.shared.providers = { get = function() return nil end, set = function() return true end }
end

function _M.clear_cache_storage()
  -- Nothing to clear in new architecture
end

function _M.setup_all()
  _M.setup_cache_shared_dicts()
  _M.setup_go_cache_http_mock()
  _M.setup_cache_env_vars()
  _M.setup_cache_diagnostics_mock()
end

function _M.reset_all()
  _M.clear_cache_storage()
end

return _M