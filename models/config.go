package models

import "time"

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
	MaxFileSize  int64
	RepoURL      string
	ProfilerPort int
	LogLevel     string
	LogFile      bool
	Whitelist    []int64
	Caching      bool

	CaptionHeader      string
	CaptionDescription string

	// Default group settings
	DefaultCaptions        bool
	DefaultSilent          bool
	DefaultNSFW            bool
	DefaultMediaGroupLimit int
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
