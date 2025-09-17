package httpserver

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"go.uber.org/mock/gomock"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"

	"go-proxy-cache/internal/cache"
	"go-proxy-cache/internal/cache/service"
	"go-proxy-cache/internal/interfaces/mock"
	"go-proxy-cache/internal/models"
	"go-proxy-cache/internal/utils"
)

// Note: Metrics are now package-level variables in the metrics package

// mockCache implements the Cache interface for testing
type mockCache struct {
	data map[string]*models.CacheEntry
	hits map[string]bool
}

func newMockCache() *mockCache {
	return &mockCache{
		data: make(map[string]*models.CacheEntry),
		hits: make(map[string]bool),
	}
}

func (m *mockCache) Get(key string) (*models.CacheEntry, bool) {
	entry, found := m.data[key]
	if found {
		m.hits[key] = true
	}
	return entry, found
}

func (m *mockCache) GetStale(key string) (*models.CacheEntry, bool) {
	entry, found := m.data[key]
	return entry, found
}

func (m *mockCache) Set(key string, val []byte, ttl models.TTL) {
	now := time.Now().Unix()
	m.data[key] = &models.CacheEntry{
		Data:      val,
		CreatedAt: now,
		StaleAt:   now + int64(ttl.Fresh.Seconds()),
		ExpiresAt: now + int64(ttl.Fresh.Seconds()) + int64(ttl.Stale.Seconds()),
	}
}

func (m *mockCache) Delete(key string) {
	delete(m.data, key)
}

// setupMockCacheClassifier configures the mock cache classifier with common expectations
func setupMockCacheClassifier(ctrl *gomock.Controller) *mock.MockCacheRulesClassifier {
	mockClassifier := mock.NewMockCacheRulesClassifier(ctrl)

	// Set up common expectations
	mockClassifier.EXPECT().GetTtl(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
		func(chain, network string, request *models.JSONRPCRequest) models.CacheInfo {
			switch request.Method {
			case "eth_getBlockByHash":
				return models.CacheInfo{TTL: 86400 * time.Second, CacheType: "permanent"}
			case "eth_blockNumber":
				return models.CacheInfo{TTL: 5 * time.Second, CacheType: "short"}
			default:
				return models.CacheInfo{TTL: 0, CacheType: "none"}
			}
		},
	).AnyTimes()

	return mockClassifier
}

// setupCacheService creates a cache service with mock caches for testing
func setupCacheService(ctrl *gomock.Controller, logger *zap.Logger) (*service.CacheService, *mockCache, *mockCache) {
	l1Cache := newMockCache()
	l2Cache := newMockCache()
	cacheClassifier := setupMockCacheClassifier(ctrl)

	// Create cache service with mocked dependencies
	cacheService := service.NewCacheService(l1Cache, l2Cache, cacheClassifier, false, logger)

	return cacheService, l1Cache, l2Cache
}

// generateCacheKey generates the actual cache key using the real key builder
func generateCacheKey(chain, network, rawBody string) (string, error) {
	keyBuilder := cache.NewKeyBuilder()
	request, err := utils.ParseJSONRPCRequest(rawBody)
	if err != nil {
		return "", err
	}
	return keyBuilder.Build(chain, network, request)
}

func TestServer_HandleGet(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	logger := zaptest.NewLogger(t)
	cacheService, l1Cache, l2Cache := setupCacheService(ctrl, logger)

	server := NewServer(cacheService, logger)

	tests := []struct {
		name           string
		requestBody    CacheRequest
		setupCache     func()
		expectedStatus int
		expectedFound  bool
	}{
		{
			name: "cache miss",
			requestBody: CacheRequest{
				Chain:   "ethereum",
				Network: "mainnet",
				RawBody: `{"method":"eth_blockNumber","params":[],"jsonrpc":"2.0","id":1}`,
			},
			expectedStatus: http.StatusOK,
			expectedFound:  false,
		},
		{
			name: "cache hit L1",
			requestBody: CacheRequest{
				Chain:   "ethereum",
				Network: "mainnet",
				RawBody: `{"method":"eth_blockNumber","params":[],"jsonrpc":"2.0","id":1}`,
			},
			setupCache: func() {
				key, _ := generateCacheKey("ethereum", "mainnet", `{"method":"eth_blockNumber","params":[],"jsonrpc":"2.0","id":1}`)
				l1Cache.Set(key, []byte(`{"result":"0x123","id":1}`), models.TTL{Fresh: time.Hour, Stale: time.Minute})
			},
			expectedStatus: http.StatusOK,
			expectedFound:  true,
		},
		{
			name: "cache hit L2",
			requestBody: CacheRequest{
				Chain:   "ethereum",
				Network: "mainnet",
				RawBody: `{"method":"eth_getBlockByHash","params":["0x123",true],"jsonrpc":"2.0","id":2}`,
			},
			setupCache: func() {
				key, _ := generateCacheKey("ethereum", "mainnet", `{"method":"eth_getBlockByHash","params":["0x123",true],"jsonrpc":"2.0","id":2}`)
				l2Cache.Set(key, []byte(`{"result":{"number":"0x123"},"id":2}`), models.TTL{Fresh: time.Hour, Stale: time.Minute})
			},
			expectedStatus: http.StatusOK,
			expectedFound:  true,
		},
		{
			name: "invalid request - missing chain",
			requestBody: CacheRequest{
				Network: "mainnet",
				RawBody: `{"method":"eth_blockNumber"}`,
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "invalid request - empty raw body",
			requestBody: CacheRequest{
				Chain:   "ethereum",
				Network: "mainnet",
				RawBody: "",
			},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset caches
			l1Cache.data = make(map[string]*models.CacheEntry)
			l2Cache.data = make(map[string]*models.CacheEntry)

			if tt.setupCache != nil {
				tt.setupCache()
			}

			body, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest("POST", "/cache/get", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			server.handleGet(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("handleGet() status = %v, want %v", w.Code, tt.expectedStatus)
			}

			if tt.expectedStatus == http.StatusOK {
				var response CacheResponse
				if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
					t.Fatalf("Failed to unmarshal response: %v", err)
				}

				if response.Found != tt.expectedFound {
					t.Errorf("handleGet() Found = %v, want %v", response.Found, tt.expectedFound)
				}

				if !response.Success {
					t.Errorf("handleGet() Success = false, want true")
				}
			}
		})
	}
}

func TestServer_HandleGet_CacheStatus(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	logger := zaptest.NewLogger(t)
	cacheService, l1Cache, l2Cache := setupCacheService(ctrl, logger)

	server := NewServer(cacheService, logger)

	tests := []struct {
		name                string
		requestBody         CacheRequest
		setupCache          func()
		expectedStatus      int
		expectedCacheStatus models.CacheStatus
	}{
		{
			name: "cache status MISS",
			requestBody: CacheRequest{
				Chain:   "ethereum",
				Network: "mainnet",
				RawBody: `{"method":"eth_blockNumber","params":[],"jsonrpc":"2.0","id":1}`,
			},
			expectedStatus:      http.StatusOK,
			expectedCacheStatus: models.CacheStatusMiss,
		},
		{
			name: "cache status HIT",
			requestBody: CacheRequest{
				Chain:   "ethereum",
				Network: "mainnet",
				RawBody: `{"method":"eth_blockNumber","params":[],"jsonrpc":"2.0","id":1}`,
			},
			setupCache: func() {
				key, _ := generateCacheKey("ethereum", "mainnet", `{"method":"eth_blockNumber","params":[],"jsonrpc":"2.0","id":1}`)
				l1Cache.Set(key, []byte(`{"result":"0x123","id":1}`), models.TTL{Fresh: time.Hour, Stale: time.Minute})
			},
			expectedStatus:      http.StatusOK,
			expectedCacheStatus: models.CacheStatusHit,
		},
		{
			name: "cache status BYPASS",
			requestBody: CacheRequest{
				Chain:   "ethereum",
				Network: "mainnet",
				RawBody: `{"method":"unknown_method","params":[],"jsonrpc":"2.0","id":1}`,
			},
			expectedStatus:      http.StatusOK,
			expectedCacheStatus: models.CacheStatusBypass,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset caches
			l1Cache.data = make(map[string]*models.CacheEntry)
			l2Cache.data = make(map[string]*models.CacheEntry)

			if tt.setupCache != nil {
				tt.setupCache()
			}

			body, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest("POST", "/cache/get", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			server.handleGet(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("handleGet() status = %v, want %v", w.Code, tt.expectedStatus)
			}

			if tt.expectedStatus == http.StatusOK {
				var response CacheResponse
				if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
					t.Fatalf("Failed to unmarshal response: %v", err)
				}

				if response.CacheStatus != tt.expectedCacheStatus {
					t.Errorf("handleGet() CacheStatus = %v, want %v", response.CacheStatus, tt.expectedCacheStatus)
				}

				if !response.Success {
					t.Errorf("handleGet() Success = false, want true")
				}
			}
		})
	}
}

func TestServer_HandleSet(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	logger := zaptest.NewLogger(t)
	cacheService, l1Cache, l2Cache := setupCacheService(ctrl, logger)

	server := NewServer(cacheService, logger)

	tests := []struct {
		name           string
		requestBody    CacheRequest
		expectedStatus int
		checkCache     bool
	}{
		{
			name: "successful set",
			requestBody: CacheRequest{
				Chain:   "ethereum",
				Network: "mainnet",
				RawBody: `{"method":"eth_getBlockByHash","params":["0x123",true],"jsonrpc":"2.0","id":1}`,
				Data:    `{"result":{"number":"0x123"},"id":1}`,
			},
			expectedStatus: http.StatusOK,
			checkCache:     true,
		},
		{
			name: "set with custom TTL",
			requestBody: CacheRequest{
				Chain:    "ethereum",
				Network:  "mainnet",
				RawBody:  `{"method":"eth_blockNumber","params":[],"jsonrpc":"2.0","id":2}`,
				Data:     `{"result":"0x456","id":2}`,
				TTL:      intPtr(3600),
				StaleTTL: intPtr(360),
			},
			expectedStatus: http.StatusOK,
			checkCache:     true,
		},
		{
			name: "invalid request - missing data",
			requestBody: CacheRequest{
				Chain:   "ethereum",
				Network: "mainnet",
				RawBody: `{"method":"eth_blockNumber"}`,
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "zero TTL method",
			requestBody: CacheRequest{
				Chain:   "ethereum",
				Network: "mainnet",
				RawBody: `{"method":"unknown_method"}`,
				Data:    `{"result":"test"}`,
			},
			expectedStatus: http.StatusOK,
			checkCache:     false, // Should not be cached due to zero TTL
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset caches
			l1Cache.data = make(map[string]*models.CacheEntry)
			l2Cache.data = make(map[string]*models.CacheEntry)

			body, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest("POST", "/cache/set", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			server.handleSet(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("handleSet() status = %v, want %v", w.Code, tt.expectedStatus)
			}

			if tt.expectedStatus == http.StatusOK {
				var response CacheResponse
				if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
					t.Fatalf("Failed to unmarshal response: %v", err)
				}

				if !response.Success {
					t.Errorf("handleSet() Success = false, want true")
				}

				// Check if data was actually cached
				if tt.checkCache {
					key, err := generateCacheKey(tt.requestBody.Chain, tt.requestBody.Network, tt.requestBody.RawBody)
					if err != nil {
						t.Fatalf("Failed to generate cache key: %v", err)
					}
					if _, found := l1Cache.data[key]; !found {
						t.Errorf("handleSet() data not found in L1 cache")
					}
					if _, found := l2Cache.data[key]; !found {
						t.Errorf("handleSet() data not found in L2 cache")
					}
				}
			}
		})
	}
}

func TestServer_HandleCacheInfo(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	logger := zaptest.NewLogger(t)
	cacheService, _, _ := setupCacheService(ctrl, logger)

	server := NewServer(cacheService, logger)

	tests := []struct {
		name           string
		requestBody    CacheRequest
		expectedStatus int
		expectedTTL    int
	}{
		{
			name: "permanent method",
			requestBody: CacheRequest{
				Chain:   "ethereum",
				Network: "mainnet",
				RawBody: `{"method":"eth_getBlockByHash","params":["0x123",true],"jsonrpc":"2.0","id":1}`,
			},
			expectedStatus: http.StatusOK,
			expectedTTL:    86400,
		},
		{
			name: "short method",
			requestBody: CacheRequest{
				Chain:   "ethereum",
				Network: "mainnet",
				RawBody: `{"method":"eth_blockNumber","params":[],"jsonrpc":"2.0","id":2}`,
			},
			expectedStatus: http.StatusOK,
			expectedTTL:    5,
		},
		{
			name: "invalid request",
			requestBody: CacheRequest{
				Chain:   "ethereum",
				Network: "mainnet",
				RawBody: "",
			},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest("POST", "/cache/info", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			server.handleCacheInfo(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("handleCacheInfo() status = %v, want %v", w.Code, tt.expectedStatus)
			}

			if tt.expectedStatus == http.StatusOK {
				var response CacheResponse
				if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
					t.Fatalf("Failed to unmarshal response: %v", err)
				}

				if !response.Success {
					t.Errorf("handleCacheInfo() Success = false, want true")
				}

				if response.TTL != tt.expectedTTL {
					t.Errorf("handleCacheInfo() TTL = %v, want %v", response.TTL, tt.expectedTTL)
				}
			}
		})
	}
}

func TestServer_HandleHealth(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	logger := zaptest.NewLogger(t)
	cacheService, _, _ := setupCacheService(ctrl, logger)

	server := NewServer(cacheService, logger)

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	server.handleHealth(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("handleHealth() status = %v, want %v", w.Code, http.StatusOK)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal health response: %v", err)
	}

	if status, ok := response["status"]; !ok || status != "healthy" {
		t.Errorf("handleHealth() status = %v, want 'healthy'", status)
	}
}

// Helper function to create int pointer
func intPtr(i int) *int {
	return &i
}
