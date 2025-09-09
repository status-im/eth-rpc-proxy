package cache_rules

import (
	"testing"
	"time"

	"go.uber.org/mock/gomock"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"

	"go-proxy-cache/internal/interfaces"
	"go-proxy-cache/internal/interfaces/mock"
	"go-proxy-cache/internal/models"
)

func TestNewClassifier(t *testing.T) {
	logger := zaptest.NewLogger(t)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConfig := mock.NewMockCacheRulesConfig(ctrl)

	classifier := NewClassifier(logger, mockConfig)

	if classifier == nil {
		t.Fatal("NewClassifier returned nil")
	}
	if classifier.logger != logger {
		t.Error("Logger not set correctly")
	}
	if classifier.configTTL != mockConfig {
		t.Error("ConfigTTL not set correctly")
	}
}

func TestNewClassifier_NilLogger(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConfig := mock.NewMockCacheRulesConfig(ctrl)

	// Should not panic with nil logger
	classifier := NewClassifier(nil, mockConfig)
	if classifier == nil {
		t.Fatal("NewClassifier returned nil with nil logger")
	}
	if classifier.logger != nil {
		t.Error("Expected nil logger to be preserved")
	}
}

func TestNewClassifier_NilConfig(t *testing.T) {
	logger := zaptest.NewLogger(t)

	// Should not panic with nil config
	classifier := NewClassifier(logger, nil)
	if classifier == nil {
		t.Fatal("NewClassifier returned nil with nil config")
	}
	if classifier.configTTL != nil {
		t.Error("Expected nil config to be preserved")
	}
}

func TestGetTtl_NilRequest(t *testing.T) {
	logger := zaptest.NewLogger(t)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConfig := mock.NewMockCacheRulesConfig(ctrl)
	classifier := NewClassifier(logger, mockConfig)

	result := classifier.GetTtl("ethereum", "mainnet", nil)

	expected := models.CacheInfo{TTL: 0, CacheType: "none"}
	if result != expected {
		t.Errorf("Expected %+v, got %+v", expected, result)
	}
}

func TestGetTtl_EmptyMethod(t *testing.T) {
	logger := zaptest.NewLogger(t)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConfig := mock.NewMockCacheRulesConfig(ctrl)
	classifier := NewClassifier(logger, mockConfig)

	request := &models.JSONRPCRequest{
		Method: "",
		ID:     1,
	}

	result := classifier.GetTtl("ethereum", "mainnet", request)

	expected := models.CacheInfo{TTL: 0, CacheType: "none"}
	if result != expected {
		t.Errorf("Expected %+v, got %+v", expected, result)
	}
}

func TestGetTtl_CacheTypeNone(t *testing.T) {
	logger := zaptest.NewLogger(t)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConfig := mock.NewMockCacheRulesConfig(ctrl)
	classifier := NewClassifier(logger, mockConfig)

	request := &models.JSONRPCRequest{
		Method: "eth_sendTransaction",
		ID:     1,
	}

	// Mock the config to return CacheTypeNone for the method
	mockConfig.EXPECT().
		GetCacheTypeForMethod("eth_sendTransaction").
		Return(models.CacheTypeNone)

	result := classifier.GetTtl("ethereum", "mainnet", request)

	expected := models.CacheInfo{TTL: 0, CacheType: models.CacheTypeNone}
	if result != expected {
		t.Errorf("Expected %+v, got %+v", expected, result)
	}
}

func TestGetTtl_ZeroTTL(t *testing.T) {
	logger := zaptest.NewLogger(t)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConfig := mock.NewMockCacheRulesConfig(ctrl)
	classifier := NewClassifier(logger, mockConfig)

	request := &models.JSONRPCRequest{
		Method: "eth_getBalance",
		ID:     1,
	}

	// Mock the config to return a cache type but zero TTL
	mockConfig.EXPECT().
		GetCacheTypeForMethod("eth_getBalance").
		Return(models.CacheTypeShort)

	mockConfig.EXPECT().
		GetTtlForCacheType("ethereum", "mainnet", models.CacheTypeShort).
		Return(time.Duration(0))

	result := classifier.GetTtl("ethereum", "mainnet", request)

	expected := models.CacheInfo{TTL: 0, CacheType: models.CacheTypeNone}
	if result != expected {
		t.Errorf("Expected %+v, got %+v", expected, result)
	}
}

func TestGetTtl_ValidCaching(t *testing.T) {
	logger := zaptest.NewLogger(t)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConfig := mock.NewMockCacheRulesConfig(ctrl)
	classifier := NewClassifier(logger, mockConfig)

	tests := []struct {
		name        string
		request     *models.JSONRPCRequest
		chain       string
		network     string
		cacheType   models.CacheType
		ttl         time.Duration
		expected    models.CacheInfo
		description string
	}{
		{
			name: "permanent cache type",
			request: &models.JSONRPCRequest{
				Method: "eth_getBlockByHash",
				ID:     1,
			},
			chain:     "ethereum",
			network:   "mainnet",
			cacheType: models.CacheTypePermanent,
			ttl:       2 * time.Hour,
			expected: models.CacheInfo{
				TTL:       2 * time.Hour,
				CacheType: models.CacheTypePermanent,
			},
			description: "should return permanent cache info for permanent cache type",
		},
		{
			name: "short cache type",
			request: &models.JSONRPCRequest{
				Method: "eth_getBalance",
				ID:     2,
			},
			chain:     "polygon",
			network:   "mainnet",
			cacheType: models.CacheTypeShort,
			ttl:       30 * time.Second,
			expected: models.CacheInfo{
				TTL:       30 * time.Second,
				CacheType: models.CacheTypeShort,
			},
			description: "should return short cache info for short cache type",
		},
		{
			name: "minimal cache type",
			request: &models.JSONRPCRequest{
				Method: "eth_call",
				ID:     3,
			},
			chain:     "arbitrum",
			network:   "one",
			cacheType: models.CacheTypeMinimal,
			ttl:       5 * time.Second,
			expected: models.CacheInfo{
				TTL:       5 * time.Second,
				CacheType: models.CacheTypeMinimal,
			},
			description: "should return minimal cache info for minimal cache type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mocks
			mockConfig.EXPECT().
				GetCacheTypeForMethod(tt.request.Method).
				Return(tt.cacheType)

			mockConfig.EXPECT().
				GetTtlForCacheType(tt.chain, tt.network, tt.cacheType).
				Return(tt.ttl)

			result := classifier.GetTtl(tt.chain, tt.network, tt.request)

			if result != tt.expected {
				t.Errorf("%s: expected %+v, got %+v", tt.description, tt.expected, result)
			}
		})
	}
}

func TestGetTtl_DifferentChainNetworkCombinations(t *testing.T) {
	logger := zaptest.NewLogger(t)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConfig := mock.NewMockCacheRulesConfig(ctrl)
	classifier := NewClassifier(logger, mockConfig)

	tests := []struct {
		name        string
		chain       string
		network     string
		description string
	}{
		{
			name:        "ethereum mainnet",
			chain:       "ethereum",
			network:     "mainnet",
			description: "should handle ethereum mainnet",
		},
		{
			name:        "polygon mainnet",
			chain:       "polygon",
			network:     "mainnet",
			description: "should handle polygon mainnet",
		},
		{
			name:        "arbitrum one",
			chain:       "arbitrum",
			network:     "one",
			description: "should handle arbitrum one",
		},
		{
			name:        "empty chain and network",
			chain:       "",
			network:     "",
			description: "should handle empty chain and network",
		},
		{
			name:        "unknown chain and network",
			chain:       "unknown",
			network:     "unknown",
			description: "should handle unknown chain and network",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := &models.JSONRPCRequest{
				Method: "eth_getBalance",
				ID:     1,
			}

			// Setup mocks
			mockConfig.EXPECT().
				GetCacheTypeForMethod("eth_getBalance").
				Return(models.CacheTypeShort)

			mockConfig.EXPECT().
				GetTtlForCacheType(tt.chain, tt.network, models.CacheTypeShort).
				Return(30 * time.Second)

			result := classifier.GetTtl(tt.chain, tt.network, request)

			expected := models.CacheInfo{
				TTL:       30 * time.Second,
				CacheType: models.CacheTypeShort,
			}

			if result != expected {
				t.Errorf("%s: expected %+v, got %+v", tt.description, expected, result)
			}
		})
	}
}

func TestGetTtl_RequestWithDifferentFields(t *testing.T) {
	logger := zaptest.NewLogger(t)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConfig := mock.NewMockCacheRulesConfig(ctrl)
	classifier := NewClassifier(logger, mockConfig)

	tests := []struct {
		name        string
		request     *models.JSONRPCRequest
		description string
	}{
		{
			name: "request with string ID",
			request: &models.JSONRPCRequest{
				Method:  "eth_getBalance",
				ID:      "test-id",
				Jsonrpc: "2.0",
			},
			description: "should handle request with string ID",
		},
		{
			name: "request with int ID",
			request: &models.JSONRPCRequest{
				Method:  "eth_getBalance",
				ID:      123,
				Jsonrpc: "2.0",
			},
			description: "should handle request with int ID",
		},
		{
			name: "request with params",
			request: &models.JSONRPCRequest{
				Method:  "eth_getBalance",
				ID:      1,
				Params:  []interface{}{"0x123", "latest"},
				Jsonrpc: "2.0",
			},
			description: "should handle request with params",
		},
		{
			name: "request without jsonrpc field",
			request: &models.JSONRPCRequest{
				Method: "eth_getBalance",
				ID:     1,
			},
			description: "should handle request without jsonrpc field",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mocks
			mockConfig.EXPECT().
				GetCacheTypeForMethod("eth_getBalance").
				Return(models.CacheTypeShort)

			mockConfig.EXPECT().
				GetTtlForCacheType("ethereum", "mainnet", models.CacheTypeShort).
				Return(30 * time.Second)

			result := classifier.GetTtl("ethereum", "mainnet", tt.request)

			expected := models.CacheInfo{
				TTL:       30 * time.Second,
				CacheType: models.CacheTypeShort,
			}

			if result != expected {
				t.Errorf("%s: expected %+v, got %+v", tt.description, expected, result)
			}
		})
	}
}

func TestGetTtl_NilConfig(t *testing.T) {
	logger := zaptest.NewLogger(t)
	classifier := NewClassifier(logger, nil)

	request := &models.JSONRPCRequest{
		Method: "eth_getBalance",
		ID:     1,
	}

	// This should panic when trying to call methods on nil config
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic when config is nil")
		}
	}()

	classifier.GetTtl("ethereum", "mainnet", request)
}

func TestClassifierInterfaceCompliance(t *testing.T) {
	// This test ensures that Classifier implements the CacheRulesClassifier interface
	logger := zaptest.NewLogger(t)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockConfig := mock.NewMockCacheRulesConfig(ctrl)
	classifier := NewClassifier(logger, mockConfig)

	// This should compile without issues if the interface is properly implemented
	var _ interfaces.CacheRulesClassifier = classifier
}

// Benchmark tests
func BenchmarkGetTtl(b *testing.B) {
	logger := zap.NewNop()
	ctrl := gomock.NewController(b)
	defer ctrl.Finish()

	mockConfig := mock.NewMockCacheRulesConfig(ctrl)
	classifier := NewClassifier(logger, mockConfig)

	request := &models.JSONRPCRequest{
		Method: "eth_getBalance",
		ID:     1,
	}

	// Setup mocks for benchmark
	mockConfig.EXPECT().
		GetCacheTypeForMethod("eth_getBalance").
		Return(models.CacheTypeShort).
		AnyTimes()

	mockConfig.EXPECT().
		GetTtlForCacheType("ethereum", "mainnet", models.CacheTypeShort).
		Return(30 * time.Second).
		AnyTimes()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		classifier.GetTtl("ethereum", "mainnet", request)
	}
}

func BenchmarkGetTtl_NilRequest(b *testing.B) {
	logger := zap.NewNop()
	ctrl := gomock.NewController(b)
	defer ctrl.Finish()

	mockConfig := mock.NewMockCacheRulesConfig(ctrl)
	classifier := NewClassifier(logger, mockConfig)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		classifier.GetTtl("ethereum", "mainnet", nil)
	}
}

func BenchmarkGetTtl_EmptyMethod(b *testing.B) {
	logger := zap.NewNop()
	ctrl := gomock.NewController(b)
	defer ctrl.Finish()

	mockConfig := mock.NewMockCacheRulesConfig(ctrl)
	classifier := NewClassifier(logger, mockConfig)

	request := &models.JSONRPCRequest{
		Method: "",
		ID:     1,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		classifier.GetTtl("ethereum", "mainnet", request)
	}
}
