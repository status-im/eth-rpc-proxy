-- keydb_mocks.lua - Mocks for KeyDB related modules and dependencies

local unpack = table.unpack or unpack -- Fix for Lua 5.4 compatibility

local _M = {}

-- Mock for resty.http module
_M.resty_http_mock = {
  new = function()
    return {
      request_uri = function(self, uri)
        -- Simple mock response
        return {
          status = 200,
          body = "mock response"
        }, nil
      end
    }
  end
}

-- Mock for lyaml module
_M.lyaml_mock = {
  load = function(content)
    if content == "valid_yaml" then
      return {
        connection = {
          connect_timeout = 200,
          send_timeout = 2000,
          read_timeout = 2000
        },
        keepalive = {
          pool_size = 20,
          max_idle_timeout = 20000
        },
        cache = {
          default_ttl = 7200,
          max_ttl = 172800
        }
      }
    elseif content == "partial_yaml" then
      return {
        connection = {
          connect_timeout = 300
        }
      }
    elseif content == "empty_sections" then
      return {}
    elseif content == "invalid_yaml" then
      error("YAML parse error")
    end
    return nil
  end
}

-- Mock for resolver_utils module
_M.resolver_utils_mock = {
  resolve_url_with_custom_dns = function(url, dns)
    if url == "redis://keydb:6379" then
      return "redis://127.0.0.1:6379", nil
    elseif url == "redis://invalid-host:6379" then
      return nil, "DNS resolution failed"
    else
      return url, nil -- Return original URL if not in mock data
    end
  end,
  parse_url = function(url)
    if not url or url == "" then
      return nil, "URL is required"
    end
    
    if url == "invalid-url" then
      return nil, "Invalid URL format - expected scheme://host:port[/path][?query]"
    end
    
    -- Simple mock parsing for test URLs
    local scheme, host, port = url:match("^(%w+)://([^:/?]+):?(%d*)")
    if not scheme or not host then
      return nil, "Invalid URL format"
    end
    
    port = port and port ~= "" and port or (scheme == "redis" and "6379" or "80")
    
    return {
      scheme = scheme,
      host = host,
      port = port,
      path = "",
      query = ""
    }
  end
}

-- Mock for resty.redis module
_M.redis_mock_factory = function()
  local redis_state = {
    connect_called = false,
    connect_host = nil,
    connect_port = nil,
    connect_result = {true, nil},
    set_timeouts_called = false,
    set_timeouts_args = {},
    setex_called = false,
    setex_args = {},
    setex_result = {true, nil},
    get_called = false,
    get_args = {},
    get_result = {nil, nil},
    set_keepalive_called = false,
    set_keepalive_args = {},
    set_keepalive_result = {true, nil}
  }
  
  return {
    state = redis_state,
    new = function()
      return {
        set_timeouts = function(self, connect_timeout, send_timeout, read_timeout)
          redis_state.set_timeouts_called = true
          redis_state.set_timeouts_args = {connect_timeout, send_timeout, read_timeout}
        end,
        connect = function(self, host, port)
          redis_state.connect_called = true
          redis_state.connect_host = host
          redis_state.connect_port = port
          return unpack(redis_state.connect_result)
        end,
        setex = function(self, key, ttl, value)
          redis_state.setex_called = true
          redis_state.setex_args = {key, ttl, value}
          return unpack(redis_state.setex_result)
        end,
        get = function(self, key)
          redis_state.get_called = true
          redis_state.get_args = {key}
          return unpack(redis_state.get_result)
        end,
        set_keepalive = function(self, max_idle_timeout, pool_size)
          redis_state.set_keepalive_called = true
          redis_state.set_keepalive_args = {max_idle_timeout, pool_size}
          return unpack(redis_state.set_keepalive_result)
        end
      }
    end
  }
end

-- Mock for keydb_config module
_M.keydb_config_mock = {
  get_keydb_url = function() return "redis://test-keydb:6379" end,
  get_connect_timeout = function() return 100 end,
  get_send_timeout = function() return 1000 end,
  get_read_timeout = function() return 1000 end,
  get_pool_size = function() return 10 end,
  get_max_idle_timeout = function() return 10000 end,
  get_default_ttl = function() return 3600 end,
  get_max_ttl = function() return 86400 end
}

-- Setup functions
function _M.setup_environment_mocks()
  -- Mock os.getenv
  _M.original_getenv = os.getenv
  os.getenv = function(key)
    if key == "KEYDB_URL" then
      return "redis://test-keydb:6379"
    elseif key == "KEYDB_CONFIG_FILE" then
      return "/test/config.yaml"
    elseif key == "CUSTOM_DNS" then
      return "127.0.0.1"
    end
    return nil
  end
end

function _M.setup_file_mocks(file_contents)
  -- Mock io.open
  _M.original_open = io.open
  io.open = function(path, mode)
    if file_contents and file_contents[path] then
      return {
        read = function(_, mode)
          if mode == "*all" then
            return file_contents[path]
          end
          return nil
        end,
        close = function() end
      }
    end
    return nil -- File not found
  end
end

function _M.setup_require_mocks(custom_mocks)
  _M.original_require = require
  _G.require = function(module_name)
    if custom_mocks and custom_mocks[module_name] then
      return custom_mocks[module_name]
    elseif module_name == "lyaml" then
      return _M.lyaml_mock
    elseif module_name == "utils.resolver_utils" then
      return _M.resolver_utils_mock
    elseif module_name == "resty.redis" then
      local redis_mock = _M.redis_mock_factory()
      return redis_mock
    elseif module_name == "resty.http" then
      return _M.resty_http_mock
    else
      return _M.original_require(module_name)
    end
  end
end

function _M.restore_all_mocks()
  if _M.original_getenv then
    os.getenv = _M.original_getenv
    _M.original_getenv = nil
  end
  
  if _M.original_open then
    io.open = _M.original_open
    _M.original_open = nil
  end
  
  if _M.original_require then
    _G.require = _M.original_require
    _M.original_require = nil
  end
end

-- Helper function to create a complete test environment for KeyDB tests
function _M.setup_keydb_test_environment(options)
  options = options or {}
  
  -- Setup environment mocks
  _M.setup_environment_mocks()
  
  -- Setup file mocks with provided file contents
  if options.file_contents then
    _M.setup_file_mocks(options.file_contents)
  end
  
  -- Setup require mocks with custom overrides
  _M.setup_require_mocks(options.custom_mocks)
  
  return {
    restore = _M.restore_all_mocks
  }
end

return _M 