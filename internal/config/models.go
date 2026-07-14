package config

import (
	"regexp"
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
	MaxFileSize  int64
	RepoURL      string
	ProfilerPort int
	MetricsPort  int
	LogLevel     zapcore.Level
	Whitelist    []int64
	Caching      bool
	// CacheVersion invalidates all previously cached media rows when
	// bumped, forcing a fresh extraction. Bump it after fixing an
	// extractor bug so stale (possibly broken) cache entries are not
	// served. Applies to every extractor, not just one.
	CacheVersion string
	Admins       []int64

	CaptionsHeader      string
	CaptionsDescription string

	DefaultCaptions        bool
	DefaultSilent          bool
	DefaultNSFW            bool
	DefaultMediaAlbumLimit int32
	DefaultLanguage        string
	DefaultDeleteLinks     bool

	AutomaticLanguageDetection bool
}

type ExtractorConfig struct {
	Proxy         string           `yaml:"proxy"`
	DownloadProxy string           `yaml:"download_proxy"`
	EdgeProxy     string           `yaml:"edge_proxy"`
	DisableProxy  bool             `yaml:"disable_proxy"`
	IgnoreRegex   []*regexp.Regexp `yaml:"ignore_regex"`
	Impersonate   bool             `yaml:"impersonate"`
	IsDisabled    bool             `yaml:"disabled"`
	Instance      []string         `yaml:"instance"`
}
