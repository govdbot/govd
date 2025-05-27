package config

import (
	"github.com/joho/godotenv"
	"go.uber.org/zap"
)

func Load() error {
	err := godotenv.Load()
	if err != nil {
		zap.S().Warn("failed to load .env file. using system env")
	}
	if err := LoadEnv(); err != nil {
		return err
	}
	if err := LoadExtractorConfigs(); err != nil {
		return err
	}
	return nil
}
