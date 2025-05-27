package models

import (
	"net/http"
	"time"
)

type DownloadConfig struct {
	ChunkSize       int               // size of each chunk in bytes
	Concurrency     int               // maximum number of concurrent downloads
	Timeout         time.Duration     // timeout for individual HTTP requests
	RetryAttempts   int               // number of retry attempts per chunk
	RetryDelay      time.Duration     // delay between retries
	Remux           bool              // whether to remux the downloaded file with ffmpeg
	ProgressUpdater func(float64)     // optional function to report download progress
	MaxInMemory     int               // maximum file size for in-memory downloads
	Headers         map[string]string // custom HTTP headers for the request
	Cookies         []*http.Cookie    // cookies to send with the request
}

func DefaultDownloadConfig() *DownloadConfig {
	return &DownloadConfig{
		ChunkSize:     10 * 1024 * 1024, // 10MB
		Concurrency:   4,
		Timeout:       30 * time.Second,
		RetryAttempts: 3,
		RetryDelay:    2 * time.Second,
		Remux:         true,
		MaxInMemory:   50 * 1024 * 1024, // 50MB
		Headers:       make(map[string]string),
		Cookies:       make([]*http.Cookie, 0),
	}
}

// GetDownloadConfig returns a new DownloadConfig with default values merged with the provided config.
// if the provided config is nil, it returns a new config with default values.
func GetDownloadConfig(config *DownloadConfig) *DownloadConfig {
	defaultConfig := DefaultDownloadConfig()
	if config == nil {
		return defaultConfig
	}
	config.Ensure()
	return config
}

func (cfg *DownloadConfig) Ensure() {
	defaultConfig := DefaultDownloadConfig()

	if cfg.ChunkSize <= 0 {
		cfg.ChunkSize = defaultConfig.ChunkSize
	}
	if cfg.Concurrency <= 0 {
		cfg.Concurrency = defaultConfig.Concurrency
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = defaultConfig.Timeout
	}
	if cfg.RetryAttempts <= 0 {
		cfg.RetryAttempts = defaultConfig.RetryAttempts
	}
	if cfg.RetryDelay <= 0 {
		cfg.RetryDelay = defaultConfig.RetryDelay
	}
	if cfg.MaxInMemory <= 0 {
		cfg.MaxInMemory = defaultConfig.MaxInMemory
	}
	if cfg.Headers == nil {
		cfg.Headers = make(map[string]string)
	}
	if cfg.Cookies == nil {
		cfg.Cookies = make([]*http.Cookie, 0)
	}
}
