package download

import (
	"context"
	"os"
	"path/filepath"

	"github.com/govdbot/govd/internal/config"
	"github.com/govdbot/govd/internal/download/chunked"
	"github.com/govdbot/govd/internal/logger"
	"github.com/govdbot/govd/internal/models"
	"github.com/govdbot/govd/internal/networking"
)

func DownloadFile(
	ctx context.Context,
	client *networking.HTTPClient,
	urlList []string,
	fileName string,
	settings *models.DownloadSettings,
) (string, error) {
	ensureDownloadDir()
	if client == nil {
		client = networking.NewHTTPClient(nil)
	}
	if settings == nil {
		settings = defaultSettings()
	}

	filePath := filepath.Join(config.Env.DownloadsDirectory, fileName)

	var lastErr error
	for _, url := range urlList {
		logger.L.Debugf("attempting download from: %s", url)

		cd, err := chunked.NewChunkedDownloader(
			ctx, client, url, settings.ChunkSize)
		if err != nil {
			lastErr = err
			continue
		}
		file, err := os.Create(filePath)
		if err != nil {
			return "", err
		}

		err = cd.Download(ctx, file, settings.NumConnections)
		file.Close()

		if err != nil {
			os.Remove(filePath)
			lastErr = err
			continue
		}

		logger.L.Infof("successfully downloaded file to: %s", filePath)
		return filePath, nil
	}

	return "", lastErr
}
