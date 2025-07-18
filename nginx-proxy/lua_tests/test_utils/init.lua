local _M = {}

-- Import nginx mocks module
local nginx_mocks = require("test_utils.nginx_mocks")

-- Test suite utilities
function _M.create_test_suite(name)
    local suite = {
        name = name,
        tests_passed = 0,
        tests_total = 0,
        tests = {}
    }
    
    function suite:add_test(test_name, test_func)
        table.insert(self.tests, {name = test_name, func = test_func})
        self.tests_total = self.tests_total + 1
        
        print("\n" .. self.tests_total .. ". Testing " .. test_name)
        
        local success, err = pcall(test_func, self)
        if success then
            self.tests_passed = self.tests_passed + 1
        else
            print("✗ Test failed: " .. test_name .. " - " .. tostring(err))
        end
    end
    
    function suite:assert_test(condition, message)
        if condition then
            print("✓ " .. message)
        else
            print("✗ " .. message)
            error("Test failed: " .. message)
        end
    end
    
    function suite:run()
        print("Testing " .. self.name)
        print(string.rep("=", #self.name + 8))
        print("\n" .. string.rep("=", #self.name + 8))
        print("Tests completed: " .. self.tests_passed .. "/" .. self.tests_total .. " passed")
        
        if self.tests_passed == self.tests_total then
            print("All tests passed! ✓")
            return true
        else
            print("Some tests failed! ✗")
            return false
        end
    end
    
    return suite
end

-- File utilities for testing
function _M.create_temp_file(content)
    local temp_name = os.tmpname()
    local file = io.open(temp_name, "w")
    if file then
        file:write(content)
        file:close()
    end
    return temp_name
end

function _M.cleanup_temp_file(filename)
    if filename then
        os.remove(filename)
    end
end

-- Nginx mocks (delegated to separate module)
function _M.setup_nginx_mocks()
    nginx_mocks.setup_all()
end

-- Setup all mocks (for backward compatibility)  
function _M.setup_all_mocks()
    _M.setup_nginx_mocks()
end

return _M 