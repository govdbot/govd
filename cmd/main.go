package main

import (
	"github.com/govdbot/govd/internal/config"
	"github.com/govdbot/govd/internal/database"
	"github.com/govdbot/govd/internal/logger"
)

func main() {
	logger.Init()
	defer logger.L.Sync()

	config.Load()

	logger.SetLevel(config.Env.LogLevel)

	if len(config.Env.Whitelist) > 0 {
		logger.L.Infof("whitelist is enabled: %v", config.Env.Whitelist)
	}

	database.Init()

	select {}
}
