package download

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"os"

	"github.com/govdbot/govd/internal/logger"
	"github.com/govdbot/govd/internal/models"
	"github.com/govdbot/govd/internal/networking"
	"github.com/govdbot/govd/internal/util/download/chunked"
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
			ctx, client, url,
			settings.ChunkSize,
		)
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
) (*bytes.Reader, error) {
	if client == nil {
		client = networking.NewHTTPClient(nil)
	}

	var lastErr error
	for _, url := range urlList {
		logger.L.Debugf("attempting download from: %s", url)
		resp, err := client.FetchWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			lastErr = err
			continue
		}

		if resp.StatusCode != 200 {
			resp.Body.Close()
			lastErr = err
			continue
		}

		data, err := io.ReadAll(resp.Body)
		if err != nil {
			resp.Body.Close()
			lastErr = err
			continue
		}

		resp.Body.Close()
		return bytes.NewReader(data), nil
	}

	return nil, lastErr
}
