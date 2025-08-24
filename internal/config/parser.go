package config

import (
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/govdbot/govd/internal/logger"
	"go.uber.org/zap/zapcore"
)

func parseEnvString(env string, dest *string, required bool) {
	if value := os.Getenv(env); value != "" {
		*dest = value
	} else if required {
		logger.L.Fatalf("%s env is not set", env)
	}
}

func parseEnvBool(env string, dest *bool, required bool) {
	if value := os.Getenv(env); value != "" {
		if parsed, err := strconv.ParseBool(value); err == nil {
			*dest = parsed
		} else {
			logger.L.Fatalf("%s env is not a valid boolean", env)
		}
	} else if required {
		logger.L.Fatalf("%s env is not set", env)
	}
}

func parseEnvInt(env string, dest *int, required bool) {
	if value := os.Getenv(env); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil {
			*dest = parsed
		} else {
			logger.L.Fatalf("%s env is not a valid integer", env)
		}
	} else if required {
		logger.L.Fatalf("%s env is not set", env)
	}
}

func parseEnvIntRange(env string, dest *int, min, max int, required bool) {
	if value := os.Getenv(env); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil {
			if parsed < min || parsed > max {
				logger.L.Fatalf("%s env must be between %d and %d", env, min, max)
			}
			*dest = parsed
		} else {
			logger.L.Fatalf("%s env is not a valid integer", env)
		}
	} else if required {
		logger.L.Fatalf("%s env is not set", env)
	}
}

func parseEnvInt64(env string, dest *int64, required bool) {
	if value := os.Getenv(env); value != "" {
		if parsed, err := strconv.ParseInt(value, 10, 64); err == nil {
			*dest = parsed
		} else {
			logger.L.Fatalf("%s env is not a valid int64", env)
		}
	} else if required {
		logger.L.Fatalf("%s env is not set", env)
	}
}

func parseEnvDuration(env string, dest *time.Duration, required bool) {
	if value := os.Getenv(env); value != "" {
		if parsed, err := time.ParseDuration(value); err == nil {
			*dest = parsed
		} else {
			logger.L.Fatalf("%s env is not a valid duration: %v", env, err)
		}
	} else if required {
		logger.L.Fatalf("%s env is not set", env)
	}
}

func parseEnvLevel(env string, dest *zapcore.Level, required bool) {
	if value := os.Getenv(env); value != "" {
		parsed, err := zapcore.ParseLevel(value)
		if err != nil {
			logger.L.Fatalf("%s env is not a valid log level: %v", env, err)
		}
		*dest = parsed
	} else if required {
		logger.L.Fatalf("%s env is not set", env)
	}
}

func parseEnvInt64Slice(env string, dest *[]int64, required bool) {
	if value := os.Getenv(env); value != "" {
		parts := strings.SplitSeq(value, ",")
		for part := range parts {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}
			id, err := strconv.ParseInt(part, 10, 64)
			if err != nil {
				logger.L.Fatalf("%s env contains an invalid int: %s", env, part)
			}
			*dest = append(*dest, id)
		}
	} else if required {
		logger.L.Fatalf("%s env is not set", env)
	}
}
