package download

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/govdbot/govd/internal/logger"
	"github.com/govdbot/govd/internal/models"
	"github.com/govdbot/govd/internal/util/download/chunked"
	"github.com/govdbot/govd/internal/util/libav"
)

func DownloadFile(
	ctx *models.ExtractorContext,
	urlList []string,
	fileName string,
	settings *models.DownloadSettings,
) (string, error) {
	if ctx == nil {
		return "", fmt.Errorf("nil extractor context")
	}
	settings = ensureDownloadSettings(settings)
	ensureDownloadDir()

	client := ctx.HTTPClient.AsDownloadClient()
	filePath := ToPath(fileName)

	// track file for cleanup
	ctx.FilesTracker.Add(&filePath)

	var lastErr error
	for _, url := range urlList {
		logger.L.Debugf("attempting download from: %s", url)

		cd, err := chunked.NewChunkedDownloader(
			ctx.Context, client, url,
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
			lastErr = err
			continue
		}

		outputPath := strings.TrimSuffix(
			filePath,
			filepath.Ext(filePath),
		) + "_remuxed" + filepath.Ext(filePath)

		err = libav.RemuxFile(filePath, outputPath)
		if err != nil {
			logger.L.Warnf("remuxing failed, using original file: %v", err)
			return filePath, nil
		}

		// replace original file with remuxed file
		os.Rename(outputPath, filePath)

		return filePath, nil
	}

	return "", lastErr
}

func DownloadFileInMemory(
	ctx *models.ExtractorContext,
	urlList []string,
) (*bytes.Reader, error) {
	if ctx == nil {
		return nil, fmt.Errorf("nil extractor context")
	}

	client := ctx.HTTPClient.AsDownloadClient()

	var lastErr error
	for _, url := range urlList {
		logger.L.Debugf("attempting download from: %s", url)
		resp, err := client.FetchWithContext(
			ctx.Context,
			http.MethodGet,
			url, nil,
		)
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
