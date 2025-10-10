package download

import (
	"bytes"
	"context"
	"os"

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
	settings = ensureDownloadSettings(settings)
	ensureDownloadDir()

	if client == nil {
		client = networking.NewHTTPClient(nil)
	}

	filePath := ToPath(fileName)

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

		return filePath, nil
	}

	return "", lastErr
}

func DownloadFileInMemory(
	ctx context.Context,
	client *networking.HTTPClient,
	urlList []string,
	settings *models.DownloadSettings,
) (*bytes.Reader, error) {
	settings = ensureDownloadSettings(settings)

	if client == nil {
		client = networking.NewHTTPClient(nil)
	}

	var lastErr error
	for _, url := range urlList {
		logger.L.Debugf("attempting download from: %s", url)

		cd, err := chunked.NewChunkedDownloader(
			ctx, client, url, settings.ChunkSize)
		if err != nil {
			lastErr = err
			continue
		}

		buffer := &bytes.Buffer{}
		err = cd.Download(ctx, buffer, settings.NumConnections)

		if err != nil {
			lastErr = err
			continue
		}

		return bytes.NewReader(buffer.Bytes()), nil
	}

	return nil, lastErr
}
