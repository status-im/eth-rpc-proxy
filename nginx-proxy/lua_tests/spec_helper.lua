-- spec_helper.lua - Setup for Busted tests

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

-- Setup cache mocks after nginx (adds shared dicts to ngx.shared)
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
    cache_mocks.clear_cache_storage() -- Only clear data, don't recreate mocks
  end,
  
  -- Setup function to be called from tests
  setup_test_environment = function()
    cache_mocks.clear_cache_storage() -- Only clear data, don't recreate mocks
  end
}

-- Note: before_each hooks should be defined in individual test files
-- because they require Busted context to be loaded 