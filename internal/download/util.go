package download

import (
	"os"
	"path/filepath"

	"github.com/govdbot/govd/internal/config"
	"github.com/govdbot/govd/internal/logger"
	"github.com/govdbot/govd/internal/models"
)

func defaultSettings() *models.DownloadSettings {
	return &models.DownloadSettings{
		NumConnections: 4,
		ChunkSize:      5 * 1024 * 1024, // 5 MB
	}
}

func ensureDownloadSettings(settings *models.DownloadSettings) *models.DownloadSettings {
	defaultSettings := defaultSettings()
	if settings == nil {
		return defaultSettings
	}
	if settings.NumConnections <= 0 {
		settings.NumConnections = defaultSettings.NumConnections
	}
	if settings.ChunkSize <= 0 {
		settings.ChunkSize = defaultSettings.ChunkSize
	}
	return settings
}

func ensureDownloadDir() {
	dir := config.Env.DownloadsDirectory
	if dir == "" {
		return
	}
	if _, err := os.Stat(dir); err != nil {
		if os.IsNotExist(err) {
			if err := os.MkdirAll(dir, 0755); err != nil {
				logger.L.Fatalf("failed to create download directory: %v", err)
			}
		} else {
			logger.L.Fatalf("failed to stat download directory: %v", err)
		}
	}
}

// constructs the full file path for a given file name
func ToPath(fileName string) string {
	return filepath.Join(config.Env.DownloadsDirectory, fileName)
}
