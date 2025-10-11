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

	HTTPSProxy string
	HTTPProxy  string
	NoProxy    string

	MaxDuration  time.Duration
	MaxFileSize  int32
	RepoURL      string
	ProfilerPort int
	LogLevel     zapcore.Level
	Whitelist    []int64
	Caching      bool

	CaptionsHeader      string
	CaptionsDescription string

	DefaultCaptions        bool
	DefaultSilent          bool
	DefaultNSFW            bool
	DefaultMediaAlbumLimit int
}

type ExtractorConfig struct {
	HTTPProxy    string `yaml:"http_proxy"`
	HTTPSProxy   string `yaml:"https_proxy"`
	NoProxy      string `yaml:"no_proxy"`
	EdgeProxyURL string `yaml:"edge_proxy_url"`
	Impersonate  bool   `yaml:"impersonate"`

	IsDisabled bool `yaml:"disabled"`

	Instance string `yaml:"instance"`
}
