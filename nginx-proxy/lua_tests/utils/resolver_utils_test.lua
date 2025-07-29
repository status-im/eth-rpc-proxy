-- resolver_utils_test.lua
describe("resolver_utils", function()
    local resolver_utils
    
    setup(function()
        -- Load spec helper which sets up all mocks
        require("spec_helper")
        resolver_utils = require("utils.resolver_utils")
    end)
    
    before_each(function()
        _G.test_helpers.setup_test_environment()
    end)

    describe("parse_url", function()
        it("should parse simple HTTP URL", function()
            local result, err = resolver_utils.parse_url("http://example.com")
            
            assert.is_nil(err)
            assert.are.equal("http", result.scheme)
            assert.are.equal("example.com", result.host)
            assert.are.equal("80", result.port)
            assert.are.equal("", result.path)
            assert.are.equal("", result.query)
        end)
        
        it("should parse HTTPS URL with custom port", function()
            local result, err = resolver_utils.parse_url("https://example.com:8443")
            
            assert.is_nil(err)
            assert.are.equal("https", result.scheme)
            assert.are.equal("example.com", result.host)
            assert.are.equal("8443", result.port)
            assert.are.equal("", result.path)
            assert.are.equal("", result.query)
        end)
        
        it("should parse Redis URL with default port", function()
            local result, err = resolver_utils.parse_url("redis://redis-server")
            
            assert.is_nil(err)
            assert.are.equal("redis", result.scheme)
            assert.are.equal("redis-server", result.host)
            assert.are.equal("6379", result.port)
            assert.are.equal("", result.path)
            assert.are.equal("", result.query)
        end)
        
        it("should parse Redis SSL URL with default port", function()
            local result, err = resolver_utils.parse_url("rediss://redis-ssl-server")
            
            assert.is_nil(err)
            assert.are.equal("rediss", result.scheme)
            assert.are.equal("redis-ssl-server", result.host)
            assert.are.equal("6380", result.port)
            assert.are.equal("", result.path)
            assert.are.equal("", result.query)
        end)
        
        it("should parse URL with path", function()
            local result, err = resolver_utils.parse_url("http://api.example.com/v1/data")
            
            assert.is_nil(err)
            assert.are.equal("http", result.scheme)
            assert.are.equal("api.example.com", result.host)
            assert.are.equal("80", result.port)
            assert.are.equal("/v1/data", result.path)
            assert.are.equal("", result.query)
        end)
        
        it("should parse URL with query parameters", function()
            local result, err = resolver_utils.parse_url("https://api.example.com/search?q=test&limit=10")
            
            assert.is_nil(err)
            assert.are.equal("https", result.scheme)
            assert.are.equal("api.example.com", result.host)
            assert.are.equal("443", result.port)
            assert.are.equal("/search", result.path)
            assert.are.equal("q=test&limit=10", result.query)
        end)
        
        it("should parse URL with path and query", function()
            local result, err = resolver_utils.parse_url("http://example.com:3000/api/v1/users?active=true")
            
            assert.is_nil(err)
            assert.are.equal("http", result.scheme)
            assert.are.equal("example.com", result.host)
            assert.are.equal("3000", result.port)
            assert.are.equal("/api/v1/users", result.path)
            assert.are.equal("active=true", result.query)
        end)
        
        it("should parse URL with only query (no path)", function()
            local result, err = resolver_utils.parse_url("http://example.com?debug=1")
            
            assert.is_nil(err)
            assert.are.equal("http", result.scheme)
            assert.are.equal("example.com", result.host)
            assert.are.equal("80", result.port)
            assert.are.equal("", result.path)
            assert.are.equal("debug=1", result.query)
        end)
        
        it("should parse URL with empty path and query", function()
            local result, err = resolver_utils.parse_url("redis://keydb.local:6380/")
            
            assert.is_nil(err)
            assert.are.equal("redis", result.scheme)
            assert.are.equal("keydb.local", result.host)
            assert.are.equal("6380", result.port)
            assert.are.equal("/", result.path)
            assert.are.equal("", result.query)
        end)
        
        it("should use default port 80 for unknown schemes", function()
            local result, err = resolver_utils.parse_url("custom://service.local")
            
            assert.is_nil(err)
            assert.are.equal("custom", result.scheme)
            assert.are.equal("service.local", result.host)
            assert.are.equal("80", result.port)
            assert.are.equal("", result.path)
            assert.are.equal("", result.query)
        end)
        
        it("should return error for nil URL", function()
            local result, err = resolver_utils.parse_url(nil)
            
            assert.is_nil(result)
            assert.are.equal("URL is required", err)
        end)
        
        it("should return error for empty URL", function()
            local result, err = resolver_utils.parse_url("")
            
            assert.is_nil(result)
            assert.are.equal("URL is required", err)
        end)
        
        it("should return error for invalid URL format - no scheme", function()
            local result, err = resolver_utils.parse_url("example.com")
            
            assert.is_nil(result)
            assert.are.equal("Invalid URL format - expected scheme://host:port[/path][?query]", err)
        end)
        
        it("should return error for invalid URL format - no host", function()
            local result, err = resolver_utils.parse_url("http://")
            
            assert.is_nil(result)
            assert.are.equal("Invalid URL format - expected scheme://host:port[/path][?query]", err)
        end)
        
        it("should return error for malformed URL", function()
            local result, err = resolver_utils.parse_url("not-a-valid-url")
            
            assert.is_nil(result)
            assert.are.equal("Invalid URL format - expected scheme://host:port[/path][?query]", err)
        end)
        
        it("should handle URLs with underscores and hyphens in hostname", function()
            local result, err = resolver_utils.parse_url("https://api-service_v2.example-domain.com:8080/test")
            
            assert.is_nil(err)
            assert.are.equal("https", result.scheme)
            assert.are.equal("api-service_v2.example-domain.com", result.host)
            assert.are.equal("8080", result.port)
            assert.are.equal("/test", result.path)
            assert.are.equal("", result.query)
        end)
        
        it("should handle complex query strings", function()
            local result, err = resolver_utils.parse_url("http://search.example.com/api?q=hello%20world&filters[]=active&filters[]=verified&sort=created_at&order=desc")
            
            assert.is_nil(err)
            assert.are.equal("http", result.scheme)
            assert.are.equal("search.example.com", result.host)
            assert.are.equal("80", result.port)
            assert.are.equal("/api", result.path)
            assert.are.equal("q=hello%20world&filters[]=active&filters[]=verified&sort=created_at&order=desc", result.query)
        end)
    end)
end)
