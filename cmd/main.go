package main

import (
	"net/http"

	"github.com/govdbot/govd/internal/bot"
	"github.com/govdbot/govd/internal/config"
	"github.com/govdbot/govd/internal/database"
	"github.com/govdbot/govd/internal/localization"
	"github.com/govdbot/govd/internal/logger"
	"github.com/govdbot/govd/internal/util"
)

func main() {
	logger.Init()
	defer logger.L.Sync()

	config.Load()
	logger.SetLevel(config.Env.LogLevel)

	if !util.CheckFFmpeg() {
		logger.L.Fatal("ffmpeg binary not found in PATH")
	}

	if len(config.Env.Whitelist) > 0 {
		logger.L.Infof("whitelist is enabled: %v", config.Env.Whitelist)
	}

	if len(config.Env.Admins) > 0 {
		logger.L.Infof("admins: %v", config.Env.Admins)
	}

	if config.Env.Profiler {
		go func() {
			logger.L.Info("starting profiler")
			if err := http.ListenAndServe("0.0.0.0:6060", nil); err != nil {
				logger.L.Fatalf("failed to start profiler: %v", err)
			}
		}()
	}

	localization.Init()
	database.Init()
	util.CleanupDownloadsJob()

	go bot.Start()

	select {}
}
