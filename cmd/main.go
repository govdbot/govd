package main

import (
	"net/http"

	_ "net/http/pprof" // profiler

	"github.com/govdbot/govd/internal/bot"
	"github.com/govdbot/govd/internal/config"
	"github.com/govdbot/govd/internal/database"
	"github.com/govdbot/govd/internal/localization"
	"github.com/govdbot/govd/internal/logger"
	"github.com/govdbot/govd/internal/util"
	"github.com/prometheus/client_golang/prometheus/promhttp"
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

	if config.Env.ProfilerPort > 0 {
		go func() {
			port := config.Env.ProfilerPort
			logger.L.Infof("starting profiler on port %d", port)
			if err := http.ListenAndServe("0.0.0.0:6060", nil); err != nil {
				logger.L.Fatalf("failed to start profiler: %v", err)
			}
		}()
	}

	if config.Env.MetricsPort > 0 {
		go func() {
			port := config.Env.MetricsPort
			logger.L.Infof("starting prometheus metrics on port %d", port)
			http.Handle("/metrics", promhttp.Handler())
			if err := http.ListenAndServe("0.0.0.0:8080", nil); err != nil {
				logger.L.Fatalf("failed to start metrics server: %v", err)
			}
		}()
	}

	localization.Init()
	database.Init()
	util.CleanupDownloadsJob()

	go bot.Start()

	select {}
}
