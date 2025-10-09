package config

import (
	"os"
	"testing"
	"time"

	"go.uber.org/zap/zapcore"
)

func TestParseEnvString(t *testing.T) {
	tests := []struct {
		name     string
		envKey   string
		envValue string
		required bool
		expected string
	}{
		{
			name:     "valid string",
			envKey:   "TEST_STRING",
			envValue: "test_value",
			required: false,
			expected: "test_value",
		},
		{
			name:     "empty string optional",
			envKey:   "TEST_STRING_EMPTY",
			envValue: "",
			required: false,
			expected: "default",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envValue != "" {
				os.Setenv(tt.envKey, tt.envValue)
				defer os.Unsetenv(tt.envKey)
			}

			result := "default"
			parseEnvString(tt.envKey, &result, tt.required)

			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestParseEnvBool(t *testing.T) {
	tests := []struct {
		name     string
		envKey   string
		envValue string
		expected bool
	}{
		{
			name:     "true",
			envKey:   "TEST_BOOL_TRUE",
			envValue: "true",
			expected: true,
		},
		{
			name:     "false",
			envKey:   "TEST_BOOL_FALSE",
			envValue: "false",
			expected: false,
		},
		{
			name:     "1",
			envKey:   "TEST_BOOL_ONE",
			envValue: "1",
			expected: true,
		},
		{
			name:     "0",
			envKey:   "TEST_BOOL_ZERO",
			envValue: "0",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Setenv(tt.envKey, tt.envValue)
			defer os.Unsetenv(tt.envKey)

			result := false
			parseEnvBool(tt.envKey, &result, false)

			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestParseEnvInt(t *testing.T) {
	tests := []struct {
		name     string
		envKey   string
		envValue string
		expected int
	}{
		{
			name:     "positive integer",
			envKey:   "TEST_INT_POSITIVE",
			envValue: "42",
			expected: 42,
		},
		{
			name:     "negative integer",
			envKey:   "TEST_INT_NEGATIVE",
			envValue: "-42",
			expected: -42,
		},
		{
			name:     "zero",
			envKey:   "TEST_INT_ZERO",
			envValue: "0",
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Setenv(tt.envKey, tt.envValue)
			defer os.Unsetenv(tt.envKey)

			result := 0
			parseEnvInt(tt.envKey, &result, false)

			if result != tt.expected {
				t.Errorf("expected %d, got %d", tt.expected, result)
			}
		})
	}
}

func TestParseEnvIntRange(t *testing.T) {
	tests := []struct {
		name     string
		envKey   string
		envValue string
		min      int
		max      int
		expected int
	}{
		{
			name:     "within range",
			envKey:   "TEST_RANGE_VALID",
			envValue: "50",
			min:      1,
			max:      100,
			expected: 50,
		},
		{
			name:     "minimum value",
			envKey:   "TEST_RANGE_MIN",
			envValue: "1",
			min:      1,
			max:      100,
			expected: 1,
		},
		{
			name:     "maximum value",
			envKey:   "TEST_RANGE_MAX",
			envValue: "100",
			min:      1,
			max:      100,
			expected: 100,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Setenv(tt.envKey, tt.envValue)
			defer os.Unsetenv(tt.envKey)

			result := 0
			parseEnvIntRange(tt.envKey, &result, tt.min, tt.max, false)

			if result != tt.expected {
				t.Errorf("expected %d, got %d", tt.expected, result)
			}
		})
	}
}

func TestParseEnvInt64(t *testing.T) {
	tests := []struct {
		name     string
		envKey   string
		envValue string
		expected int64
	}{
		{
			name:     "small int64",
			envKey:   "TEST_INT64_SMALL",
			envValue: "123",
			expected: 123,
		},
		{
			name:     "large int64",
			envKey:   "TEST_INT64_LARGE",
			envValue: "9223372036854775807",
			expected: 9223372036854775807,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Setenv(tt.envKey, tt.envValue)
			defer os.Unsetenv(tt.envKey)

			result := int64(0)
			parseEnvInt64(tt.envKey, &result, false)

			if result != tt.expected {
				t.Errorf("expected %d, got %d", tt.expected, result)
			}
		})
	}
}

func TestParseEnvDuration(t *testing.T) {
	tests := []struct {
		name     string
		envKey   string
		envValue string
		expected time.Duration
	}{
		{
			name:     "seconds",
			envKey:   "TEST_DURATION_SECONDS",
			envValue: "30s",
			expected: 30 * time.Second,
		},
		{
			name:     "minutes",
			envKey:   "TEST_DURATION_MINUTES",
			envValue: "5m",
			expected: 5 * time.Minute,
		},
		{
			name:     "hours",
			envKey:   "TEST_DURATION_HOURS",
			envValue: "2h",
			expected: 2 * time.Hour,
		},
		{
			name:     "combined",
			envKey:   "TEST_DURATION_COMBINED",
			envValue: "1h30m",
			expected: 90 * time.Minute,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Setenv(tt.envKey, tt.envValue)
			defer os.Unsetenv(tt.envKey)

			result := time.Duration(0)
			parseEnvDuration(tt.envKey, &result, false)

			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestParseEnvLevel(t *testing.T) {
	tests := []struct {
		name     string
		envKey   string
		envValue string
		expected zapcore.Level
	}{
		{
			name:     "debug level",
			envKey:   "TEST_LEVEL_DEBUG",
			envValue: "debug",
			expected: zapcore.DebugLevel,
		},
		{
			name:     "info level",
			envKey:   "TEST_LEVEL_INFO",
			envValue: "info",
			expected: zapcore.InfoLevel,
		},
		{
			name:     "warn level",
			envKey:   "TEST_LEVEL_WARN",
			envValue: "warn",
			expected: zapcore.WarnLevel,
		},
		{
			name:     "error level",
			envKey:   "TEST_LEVEL_ERROR",
			envValue: "error",
			expected: zapcore.ErrorLevel,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Setenv(tt.envKey, tt.envValue)
			defer os.Unsetenv(tt.envKey)

			result := zapcore.InfoLevel
			parseEnvLevel(tt.envKey, &result, false)

			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}
