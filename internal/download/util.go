package download

import (
	"os"

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
