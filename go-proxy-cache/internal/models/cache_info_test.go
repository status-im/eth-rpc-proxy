package models

import (
	"testing"
)

func TestCacheLevelFromIndex(t *testing.T) {
	tests := []struct {
		name     string
		index    int
		expected CacheLevel
	}{
		{
			name:     "Index 0 returns L1",
			index:    0,
			expected: CacheLevelL1,
		},
		{
			name:     "Index 1 returns L2",
			index:    1,
			expected: CacheLevelL2,
		},
		{
			name:     "Index 2 returns L3",
			index:    2,
			expected: CacheLevel("L3"),
		},
		{
			name:     "Index 3 returns L4",
			index:    3,
			expected: CacheLevel("L4"),
		},
		{
			name:     "Index 10 returns L11",
			index:    10,
			expected: CacheLevel("L11"),
		},
		{
			name:     "Negative index -1 returns L1 (fallback)",
			index:    -1,
			expected: CacheLevelL1,
		},
		{
			name:     "Negative index -5 returns L1 (fallback)",
			index:    -5,
			expected: CacheLevelL1,
		},
		{
			name:     "Large negative index returns L1 (fallback)",
			index:    -100,
			expected: CacheLevelL1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CacheLevelFromIndex(tt.index)
			if result != tt.expected {
				t.Errorf("CacheLevelFromIndex(%d) = %v, want %v", tt.index, result, tt.expected)
			}
		})
	}
}

func TestCacheLevel_String(t *testing.T) {
	tests := []struct {
		name     string
		level    CacheLevel
		expected string
	}{
		{
			name:     "L1 string representation",
			level:    CacheLevelL1,
			expected: "L1",
		},
		{
			name:     "L2 string representation",
			level:    CacheLevelL2,
			expected: "L2",
		},
		{
			name:     "MISS string representation",
			level:    CacheLevelMiss,
			expected: "MISS",
		},
		{
			name:     "Custom level string representation",
			level:    CacheLevel("L5"),
			expected: "L5",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.level.String()
			if result != tt.expected {
				t.Errorf("CacheLevel.String() = %v, want %v", result, tt.expected)
			}
		})
	}
}
