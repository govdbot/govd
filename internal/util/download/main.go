package download

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"github.com/govdbot/govd/internal/logger"
	"github.com/govdbot/govd/internal/models"
	"github.com/govdbot/govd/internal/util/download/chunked"
	"github.com/govdbot/govd/internal/util/download/segmented"
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
			ctx.Context, client, url, settings,
		)
		if err != nil {
			lastErr = err
			continue
		}
		file, err := os.Create(filePath)
		if err != nil {
			return "", err
		}
		defer file.Close()

		err = cd.Download(ctx, file, settings.NumConnections)

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

func DownloadFileWithSegments(
	ctx *models.ExtractorContext,
	initSegmentURL string,
	segmentURLs []string,
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

	tempDir := ToPath("segments" + uuid.NewString()[:8])

	// track temp dir for cleanup
	ctx.FilesTracker.Add(&tempDir)

	if err := os.MkdirAll(tempDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create temporary directory: %w", err)
	}

	logger.L.Debugf("attempting download from: %s", segmentURLs[0])

	sd := segmented.NewSegmentedDownloader(
		ctx.Context, client,
		tempDir, segmentURLs,
		&segmented.SegmentedDownloaderOptions{
			InitSegment:   initSegmentURL,
			DecryptionKey: settings.DecryptionKey,
			Retries:       settings.Retries,
		},
	)

	file, err := os.Create(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	err = sd.Download(ctx.Context, file, settings.NumConnections)
	if err != nil {
		return "", err
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

func DownloadFileInMemory(
	ctx *models.ExtractorContext,
	urlList []string,
	settings *models.DownloadSettings,
) (*bytes.Reader, error) {
	if ctx == nil {
		return nil, fmt.Errorf("nil extractor context")
	}

	client := ctx.HTTPClient.AsDownloadClient()
	maxRetries := max(settings.Retries, 1)

	for _, url := range urlList {
		for attempt := range maxRetries {
			logger.L.Debugf("attempting download from: %s (attempt %d/%d)", url, attempt+1, maxRetries)
			resp, err := client.FetchWithContext(
				ctx.Context,
				http.MethodGet,
				url, nil,
			)
			if err != nil {
				continue
			}

			if resp.StatusCode != 200 {
				resp.Body.Close()
				continue
			}

			data, err := io.ReadAll(resp.Body)
			if err != nil {
				resp.Body.Close()
				continue
			}

			resp.Body.Close()
			return bytes.NewReader(data), nil
		}
	}

	return nil, fmt.Errorf("all download attempts failed")
}
