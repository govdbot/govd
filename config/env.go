package config

import (
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/govdbot/govd/models"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
	"go.uber.org/zap"
)

var Env = GetDefaultConfig()

func LoadEnv() error {
	if value := os.Getenv("DB_HOST"); value != "" {
		Env.DBHost = value
	} else {
		zap.S().Fatalf("DB_HOST env is not set")
	}
	if value := os.Getenv("DB_PORT"); value != "" {
		if port, err := strconv.Atoi(value); err == nil {
			Env.DBPort = port
		} else {
			zap.S().Fatal("DB_PORT env is not a valid integer")
		}
	} else {
		zap.S().Fatalf("DB_PORT env is not set")
	}
	if value := os.Getenv("DB_NAME"); value != "" {
		Env.DBName = value
	} else {
		zap.S().Fatal("DB_NAME env is not set")
	}
	if value := os.Getenv("DB_USER"); value != "" {
		Env.DBUser = value
	} else {
		zap.S().Fatalf("DB_USER env is not set")
	}
	if value := os.Getenv("DB_PASSWORD"); value != "" {
		Env.DBPassword = value
	} else {
		zap.S().Fatalf("DB_PASSWORD env is not set")
	}
	if value := os.Getenv("BOT_TOKEN"); value != "" {
		Env.BotToken = value
	} else {
		zap.S().Fatalf("BOT_TOKEN env is not set")
	}
	if value := os.Getenv("BOT_API_URL"); value != "" {
		Env.BotAPIURL = value
	} else {
		zap.S().Warnf("BOT_API_URL is not set, using default %s", Env.BotAPIURL)
	}
	if value := os.Getenv("CONCURRENT_UPDATES"); value != "" {
		if updates, err := strconv.Atoi(value); err == nil {
			Env.ConcurrentUpdates = updates
		} else {
			zap.S().Fatal("CONCURRENT_UPDATES env is not a valid integer")
		}
	} else {
		zap.S().Warnf("CONCURRENT_UPDATES is not set, using default %d", Env.ConcurrentUpdates)
	}
	if value := os.Getenv("DOWNLOADS_DIR"); value != "" {
		Env.DownloadsDirectory = value
	} else {
		zap.S().Warnf("DOWNLOADS_DIR is not set, using default %s", Env.DownloadsDirectory)
	}
	if value := os.Getenv("HTTP_PROXY"); value != "" {
		Env.HTTPProxy = value
	}
	if value := os.Getenv("HTTPS_PROXY"); value != "" {
		Env.HTTPSProxy = value
	}
	if value := os.Getenv("NO_PROXY"); value != "" {
		Env.NoProxy = value
	}
	if value := os.Getenv("MAX_DURATION"); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			Env.MaxDuration = duration
		} else {
			zap.S().Fatalf("MAX_DURATION env is not a valid duration: %v", err)
		}
	}
	if value := os.Getenv("MAX_FILE_SIZE"); value != "" {
		if size, err := strconv.Atoi(value); err == nil {
			Env.MaxFileSize = int64(size) * 1024 * 1024
		}
	}
	if value := os.Getenv("REPO_URL"); value != "" {
		Env.RepoURL = value
	}
	if value := os.Getenv("PROFILER_PORT"); value != "" {
		if port, err := strconv.Atoi(value); err == nil {
			Env.ProfilerPort = port
		} else {
			zap.S().Fatal("PROFILER_PORT env is not a valid integer")
		}
	}
	if value := os.Getenv("LOG_LEVEL"); value != "" {
		Env.LogLevel = value
	}
	if value := os.Getenv("LOG_FILE"); value != "" {
		if logFile, err := strconv.ParseBool(value); err == nil {
			Env.LogFile = logFile
		} else {
			zap.S().Fatal("LOG_FILE env is not a valid boolean")
		}
	}
	if value := os.Getenv("WHITELIST"); value != "" {
		parts := strings.Split(value, ",")
		for _, part := range parts {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}
			id, err := strconv.ParseInt(part, 10, 64)
			if err != nil {
				zap.S().Fatalf("WHITELIST env contains an invalid ID: %s", part)
			}
			if part != "" {
				Env.Whitelist = append(Env.Whitelist, id)
			}
		}
	}
	if value := os.Getenv("CACHING"); value != "" {
		if caching, err := strconv.ParseBool(value); err == nil {
			Env.Caching = caching
		} else {
			zap.S().Fatal("CACHING env is not a valid boolean")
		}
	} else {
		zap.S().Warnf("CACHING is not set, using default %t", Env.Caching)
	}
	if value := os.Getenv("CAPTION_HEADER"); value != "" {
		Env.CaptionHeader = value
	}
	if value := os.Getenv("CAPTION_DESCRIPTION"); value != "" {
		Env.CaptionDescription = value
	}
	if value := os.Getenv("DEFAULT_CAPTIONS"); value != "" {
		if defaultCaptions, err := strconv.ParseBool(value); err == nil {
			Env.DefaultCaptions = defaultCaptions
		} else {
			zap.S().Fatal("DEFAULT_CAPTIONS env is not a valid boolean")
		}
	} else {
		zap.S().Warnf("DEFAULT_CAPTIONS is not set, using default %t", Env.DefaultCaptions)
	}
	if value := os.Getenv("DEFAULT_SILENT"); value != "" {
		if defaultSilent, err := strconv.ParseBool(value); err == nil {
			Env.DefaultSilent = defaultSilent
		} else {
			zap.S().Fatal("DEFAULT_SILENT env is not a valid boolean")
		}
	} else {
		zap.S().Warnf("DEFAULT_SILENT is not set, using default %t", Env.DefaultSilent)
	}
	if value := os.Getenv("DEFAULT_ENABLE_NSFW"); value != "" {
		if defaultNSFW, err := strconv.ParseBool(value); err == nil {
			Env.DefaultNSFW = defaultNSFW
		} else {
			zap.S().Fatal("DEFAULT_ENABLE_NSFW env is not a valid boolean")
		}
	} else {
		zap.S().Warnf("DEFAULT_ENABLE_NSFW is not set, using default %t", Env.DefaultNSFW)
	}
	if value := os.Getenv("DEFAULT_MEDIA_LIMIT"); value != "" {
		if defaultMediaLimit, err := strconv.Atoi(value); err == nil {
			if defaultMediaLimit >= 1 && defaultMediaLimit <= 20 {
				Env.DefaultMediaGroupLimit = defaultMediaLimit
			} else {
				zap.S().Fatal("DEFAULT_MEDIA_LIMIT must be between 1 and 20")
			}
		} else {
			zap.S().Fatal("DEFAULT_MEDIA_LIMIT env is not a valid integer")
		}
	} else {
		zap.S().Warnf("DEFAULT_MEDIA_LIMIT is not set, using default %d", Env.DefaultMediaGroupLimit)
	}
	return nil
}

func GetDefaultConfig() *models.EnvConfig {
	return &models.EnvConfig{
		DBHost: "localhost",
		DBPort: 3306,
		DBName: "govd",
		DBUser: "govd",

		BotAPIURL:         gotgbot.DefaultAPIURL,
		ConcurrentUpdates: ext.DefaultMaxRoutines,

		DownloadsDirectory: "downloads",

		MaxDuration: time.Hour,
		MaxFileSize: 1000 * 1024 * 1024, // 1GB
		RepoURL:     "https://github.com/govdbot/govd",
		LogLevel:    "info",
		Caching:     true,

		CaptionHeader:      "<a href='{{url}}'>source</a> - @govd_bot",
		CaptionDescription: "<blockquote expandable>{{text}}</blockquote>",

		// Default group settings
		DefaultCaptions:        false,
		DefaultSilent:          false,
		DefaultNSFW:            false,
		DefaultMediaGroupLimit: 10,
	}
}
