package util

import (
	"regexp"
	"testing"
)

func TestChunkedSlice(t *testing.T) {
	tests := []struct {
		name     string
		slice    []int
		size     int
		expected [][]int
	}{
		{
			name:     "empty slice",
			slice:    []int{},
			size:     2,
			expected: [][]int{},
		},
		{
			name:     "exact chunks",
			slice:    []int{1, 2, 3, 4, 5, 6},
			size:     2,
			expected: [][]int{{1, 2}, {3, 4}, {5, 6}},
		},
		{
			name:     "uneven chunks",
			slice:    []int{1, 2, 3, 4, 5},
			size:     2,
			expected: [][]int{{1, 2}, {3, 4}, {5}},
		},
		{
			name:     "size larger than slice",
			slice:    []int{1, 2, 3},
			size:     10,
			expected: [][]int{{1, 2, 3}},
		},
		{
			name:     "size of 1",
			slice:    []int{1, 2, 3},
			size:     1,
			expected: [][]int{{1}, {2}, {3}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ChunkedSlice(tt.slice, tt.size)
			if len(result) != len(tt.expected) {
				t.Fatalf("expected %d chunks, got %d", len(tt.expected), len(result))
			}
			for i := range result {
				if len(result[i]) != len(tt.expected[i]) {
					t.Fatalf("chunk %d: expected length %d, got %d", i, len(tt.expected[i]), len(result[i]))
				}
				for j := range result[i] {
					if result[i][j] != tt.expected[i][j] {
						t.Fatalf("chunk %d, element %d: expected %d, got %d", i, j, tt.expected[i][j], result[i][j])
					}
				}
			}
		})
	}
}

func TestGetNamedGroups(t *testing.T) {
	tests := []struct {
		name     string
		pattern  string
		str      string
		expected map[string]string
	}{
		{
			name:    "simple capture",
			pattern: `(?P<id>\d+)`,
			str:     "12345",
			expected: map[string]string{
				"id":    "12345",
				"match": "12345",
			},
		},
		{
			name:    "multiple captures",
			pattern: `(?P<protocol>https?)://(?P<domain>[^/]+)/(?P<path>.*)`,
			str:     "https://example.com/path/to/resource",
			expected: map[string]string{
				"protocol": "https",
				"domain":   "example.com",
				"path":     "path/to/resource",
				"match":    "https://example.com/path/to/resource",
			},
		},
		{
			name:    "no match",
			pattern: `(?P<id>\d+)`,
			str:     "abc",
			expected: map[string]string{
				"match": "",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			re := regexp.MustCompile(tt.pattern)
			result := GetNamedGroups(re, tt.str)

			for key, expectedValue := range tt.expected {
				if result[key] != expectedValue {
					t.Errorf("expected %s=%q, got %q", key, expectedValue, result[key])
				}
			}
		})
	}
}

func TestExtractBaseHost(t *testing.T) {
	tests := []struct {
		name        string
		url         string
		expected    string
		expectError bool
	}{
		{
			name:        "simple domain",
			url:         "https://example.com/path",
			expected:    "example",
			expectError: false,
		},
		{
			name:        "subdomain",
			url:         "https://www.example.com/path",
			expected:    "example",
			expectError: false,
		},
		{
			name:        "multiple subdomains",
			url:         "https://api.v2.example.com/path",
			expected:    "example",
			expectError: false,
		},
		{
			name:        "co.uk domain",
			url:         "https://example.co.uk/path",
			expected:    "example",
			expectError: false,
		},
		{
			name:        "tiktok",
			url:         "https://www.tiktok.com/@user/video/123456",
			expected:    "tiktok",
			expectError: false,
		},
		{
			name:        "invalid url",
			url:         "://invalid",
			expected:    "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ExtractBaseHost(tt.url)
			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if result != tt.expected {
					t.Errorf("expected %q, got %q", tt.expected, result)
				}
			}
		})
	}
}
