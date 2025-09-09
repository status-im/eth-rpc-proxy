-- spec_helper.lua - Simplified setup for go-proxy-cache integration tests

-- Setup LUA_PATH to include our modules
local current_dir = debug.getinfo(1, "S").source:sub(2):match("(.*)/")
local base_dir = current_dir .. "/.."

package.path = package.path .. ";" .. 
  base_dir .. "/lua/?.lua;" ..
  base_dir .. "/lua/?/init.lua;" ..
  base_dir .. "/lua/cache/?.lua;" ..
  base_dir .. "/lua/auth/?.lua;" ..
  base_dir .. "/lua/providers/?.lua;" ..
  base_dir .. "/lua/utils/?.lua;" ..
  current_dir .. "/?.lua;" ..
  current_dir .. "/?/init.lua"

-- Setup nginx mocks first (creates ngx global)
local nginx_mocks = require("test_utils.nginx_mocks")
nginx_mocks.setup_all()

-- Setup simplified cache mocks for go-proxy-cache integration
local cache_mocks = require("test_utils.cache_mocks")
cache_mocks.setup_all()

-- Global helper functions for tests
_G.test_helpers = {
  create_temp_file = function(content)
    local temp_name = os.tmpname()
    local file = io.open(temp_name, "w")
    if file then
      file:write(content)
      file:close()
    end
    return temp_name
  end,
  
  cleanup_temp_file = function(filename)
    if filename then
      os.remove(filename)
    end
  end,
  
  reset_mocks = function()
    cache_mocks.clear_cache_storage()
  end,
  
  setup_test_environment = function()
    cache_mocks.clear_cache_storage()
  end,
  
  -- Helper to mock go-proxy-cache responses
  mock_cache_response = function(response_data)
    cache_mocks.setup_go_cache_http_mock(response_data)
  end
}