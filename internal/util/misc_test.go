package util

import (
	"regexp"
	"testing"
)

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
