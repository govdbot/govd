package config

import (
	"time"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
	"go.uber.org/zap"
)

var Env = GetDefaultConfig()

func loadEnv() {
	parseEnvString("DB_HOST", &Env.DBHost, false)
	parseEnvInt("DB_PORT", &Env.DBPort, false)
	parseEnvString("DB_NAME", &Env.DBName, false)
	parseEnvString("DB_USER", &Env.DBUser, false)
	parseEnvString("DB_PASSWORD", &Env.DBPassword, false)
	parseEnvString("BOT_TOKEN", &Env.BotToken, true)
	parseEnvString("BOT_API_URL", &Env.BotAPIURL, false)
	parseEnvInt("CONCURRENT_UPDATES", &Env.ConcurrentUpdates, false)
	parseEnvString("DOWNLOADS_DIR", &Env.DownloadsDirectory, false)
	parseEnvString("HTTP_PROXY", &Env.HTTPProxy, false)
	parseEnvString("HTTPS_PROXY", &Env.HTTPSProxy, false)
	parseEnvString("NO_PROXY", &Env.NoProxy, false)
	parseEnvDuration("MAX_DURATION", &Env.MaxDuration, false)
	parseEnvInt32("MAX_FILE_SIZE", &Env.MaxFileSize, false)
	parseEnvString("REPO_URL", &Env.RepoURL, false)
	parseEnvInt("PROFILER_PORT", &Env.ProfilerPort, false)
	parseEnvLevel("LOG_LEVEL", &Env.LogLevel, false)
	parseEnvInt64Slice("WHITELIST", &Env.Whitelist, false)
	parseEnvBool("CACHING", &Env.Caching, false)
	parseEnvString("CAPTIONS_HEADER", &Env.CaptionsHeader, false)
	parseEnvString("CAPTIONS_DESCRIPTION", &Env.CaptionsDescription, false)
	parseEnvBool("DEFAULT_ENABLE_CAPTIONS", &Env.DefaultCaptions, false)
	parseEnvBool("DEFAULT_ENABLE_SILENT", &Env.DefaultSilent, false)
	parseEnvBool("DEFAULT_ENABLE_NSFW", &Env.DefaultNSFW, false)
	parseEnvIntRange("DEFAULT_MEDIA_ALBUM_LIMIT", &Env.DefaultMediaAlbumLimit, 1, 20, false)
}

func GetDefaultConfig() *EnvConfig {
	return &EnvConfig{
		DBHost: "localhost",
		DBPort: 5432,
		DBName: "govd",
		DBUser: "govd",

		BotAPIURL:         gotgbot.DefaultAPIURL,
		ConcurrentUpdates: ext.DefaultMaxRoutines,

		DownloadsDirectory: "downloads",

		MaxDuration: time.Hour,
		MaxFileSize: 1000 * 1024 * 1024, // 1GB
		RepoURL:     "https://github.com/govdbot/govd",
		LogLevel:    zap.InfoLevel,
		Caching:     true,

		CaptionsHeader:      "<a href='{{url}}'>source</a> - @govd_bot",
		CaptionsDescription: "<blockquote expandable>{{text}}</blockquote>",

		DefaultCaptions:        false,
		DefaultSilent:          false,
		DefaultNSFW:            false,
		DefaultMediaAlbumLimit: 10,
	}
}
