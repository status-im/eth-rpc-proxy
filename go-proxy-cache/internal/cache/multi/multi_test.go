package multi

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap"

	"go-proxy-cache/internal/interfaces"
	"go-proxy-cache/internal/interfaces/mock"
	"go-proxy-cache/internal/models"
)

func TestNewMultiCache(t *testing.T) {
	logger := zap.NewNop()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	cache1 := mock.NewMockCache(ctrl)
	cache2 := mock.NewMockCache(ctrl)
	caches := []interfaces.Cache{cache1, cache2}

	multiCache := NewMultiCache(caches, logger)

	assert.NotNil(t, multiCache)
	mc := multiCache.(*MultiCache)
	assert.Equal(t, 2, len(mc.caches))
	assert.Equal(t, cache1, mc.caches[0])
	assert.Equal(t, cache2, mc.caches[1])
}

func TestMultiCache_Get_FirstCacheHit(t *testing.T) {
	logger := zap.NewNop()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	cache1 := mock.NewMockCache(ctrl)
	cache2 := mock.NewMockCache(ctrl)
	caches := []interfaces.Cache{cache1, cache2}

	multiCache := NewMultiCache(caches, logger)

	expectedVal := []byte("test-value")
	cache1.EXPECT().Get("test-key").Return(expectedVal, true, true).Times(1)
	// cache2.Get should not be called since cache1 has the value

	val, fresh, found := multiCache.Get("test-key")

	assert.True(t, found)
	assert.True(t, fresh)
	assert.Equal(t, expectedVal, val)
}

func TestMultiCache_Get_SecondCacheHit(t *testing.T) {
	logger := zap.NewNop()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	cache1 := mock.NewMockCache(ctrl)
	cache2 := mock.NewMockCache(ctrl)
	caches := []interfaces.Cache{cache1, cache2}

	multiCache := NewMultiCache(caches, logger)

	expectedVal := []byte("test-value")

	cache1.EXPECT().Get("test-key").Return(nil, false, false).Times(1)
	cache2.EXPECT().Get("test-key").Return(expectedVal, true, true).Times(1)

	val, fresh, found := multiCache.Get("test-key")

	assert.True(t, found)
	assert.True(t, fresh)
	assert.Equal(t, expectedVal, val)
}

func TestMultiCache_Get_AllCachesMiss(t *testing.T) {
	logger := zap.NewNop()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	cache1 := mock.NewMockCache(ctrl)
	cache2 := mock.NewMockCache(ctrl)
	caches := []interfaces.Cache{cache1, cache2}

	multiCache := NewMultiCache(caches, logger)

	cache1.EXPECT().Get("test-key").Return(nil, false, false).Times(1)
	cache2.EXPECT().Get("test-key").Return(nil, false, false).Times(1)

	val, fresh, found := multiCache.Get("test-key")

	assert.False(t, found)
	assert.False(t, fresh)
	assert.Nil(t, val)
}

func TestMultiCache_Get_NoCaches(t *testing.T) {
	logger := zap.NewNop()

	multiCache := NewMultiCache([]interfaces.Cache{}, logger)

	val, fresh, found := multiCache.Get("test-key")

	assert.False(t, found)
	assert.False(t, fresh)
	assert.Nil(t, val)
}

func TestMultiCache_GetStale_FirstCacheHit(t *testing.T) {
	logger := zap.NewNop()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	cache1 := mock.NewMockCache(ctrl)
	cache2 := mock.NewMockCache(ctrl)
	caches := []interfaces.Cache{cache1, cache2}

	multiCache := NewMultiCache(caches, logger)

	expectedVal := []byte("test-value")
	cache1.EXPECT().GetStale("test-key").Return(expectedVal, true).Times(1)

	val, found := multiCache.GetStale("test-key")

	assert.True(t, found)
	assert.Equal(t, expectedVal, val)
}

func TestMultiCache_GetStale_SecondCacheHit(t *testing.T) {
	logger := zap.NewNop()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	cache1 := mock.NewMockCache(ctrl)
	cache2 := mock.NewMockCache(ctrl)
	caches := []interfaces.Cache{cache1, cache2}

	multiCache := NewMultiCache(caches, logger)

	expectedVal := []byte("test-value")

	cache1.EXPECT().GetStale("test-key").Return(nil, false).Times(1)
	cache2.EXPECT().GetStale("test-key").Return(expectedVal, true).Times(1)

	val, found := multiCache.GetStale("test-key")

	assert.True(t, found)
	assert.Equal(t, expectedVal, val)
}

func TestMultiCache_Set_AllCaches(t *testing.T) {
	logger := zap.NewNop()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	cache1 := mock.NewMockCache(ctrl)
	cache2 := mock.NewMockCache(ctrl)
	caches := []interfaces.Cache{cache1, cache2}

	multiCache := NewMultiCache(caches, logger)

	testVal := []byte("test-value")
	testTTL := models.TTL{Fresh: 60 * time.Second, Stale: 30 * time.Second}

	cache1.EXPECT().Set("test-key", testVal, testTTL).Times(1)
	cache2.EXPECT().Set("test-key", testVal, testTTL).Times(1)

	multiCache.Set("test-key", testVal, testTTL)
}

func TestMultiCache_Set_NoCaches(t *testing.T) {
	logger := zap.NewNop()

	multiCache := NewMultiCache([]interfaces.Cache{}, logger)

	testVal := []byte("test-value")
	testTTL := models.TTL{Fresh: 60 * time.Second, Stale: 30 * time.Second}

	// Should not panic
	multiCache.Set("test-key", testVal, testTTL)
}

func TestMultiCache_Delete_AllCaches(t *testing.T) {
	logger := zap.NewNop()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	cache1 := mock.NewMockCache(ctrl)
	cache2 := mock.NewMockCache(ctrl)
	caches := []interfaces.Cache{cache1, cache2}

	multiCache := NewMultiCache(caches, logger)

	cache1.EXPECT().Delete("test-key").Times(1)
	cache2.EXPECT().Delete("test-key").Times(1)

	multiCache.Delete("test-key")
}

func TestMultiCache_Delete_NoCaches(t *testing.T) {
	logger := zap.NewNop()

	multiCache := NewMultiCache([]interfaces.Cache{}, logger)

	// Should not panic
	multiCache.Delete("test-key")
}

func TestMultiCache_GetCacheCount(t *testing.T) {
	logger := zap.NewNop()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	cache1 := mock.NewMockCache(ctrl)
	cache2 := mock.NewMockCache(ctrl)
	caches := []interfaces.Cache{cache1, cache2}

	multiCache := NewMultiCache(caches, logger)
	mc := multiCache.(*MultiCache)

	assert.Equal(t, 2, mc.GetCacheCount())
}
