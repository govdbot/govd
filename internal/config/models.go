package config

import (
	"time"

	"go.uber.org/zap/zapcore"
)

type EnvConfig struct {
	DBHost     string
	DBPort     int
	DBName     string
	DBUser     string
	DBPassword string

	BotAPIURL         string
	BotToken          string
	ConcurrentUpdates int

	DownloadsDirectory string

	Proxy string

	MaxDuration  time.Duration
	MaxFileSize  int32
	RepoURL      string
	ProfilerPort int
	LogLevel     zapcore.Level
	Whitelist    []int64
	Caching      bool
	Admins       []int64

	CaptionsHeader      string
	CaptionsDescription string

	DefaultCaptions        bool
	DefaultSilent          bool
	DefaultNSFW            bool
	DefaultMediaAlbumLimit int
}

type ExtractorConfig struct {
	Proxy         string `yaml:"proxy"`
	DownloadProxy string `yaml:"download_proxy"`
	EdgeProxy     string `yaml:"edge_proxy"`
	DisableProxy  bool   `yaml:"disable_proxy"`

	Impersonate bool `yaml:"impersonate"`

	IsDisabled bool `yaml:"disabled"`

	Instance []string `yaml:"instance"`
}
