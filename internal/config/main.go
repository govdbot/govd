package config

import (
	"github.com/govdbot/govd/internal/logger"
	"github.com/joho/godotenv"
)

func Load() {
	err := godotenv.Load()
	if err != nil {
		logger.L.Warn("failed to load .env file. using system env")
	}
	loadEnv()
}
