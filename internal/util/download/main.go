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
	"github.com/govdbot/govd/internal/networking"
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
	ctx.FilesTracker.Add(filePath)

	file, err := os.Create(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	var lastErr error
	for _, url := range urlList {
		ctx.Debugf("attempting download from: %s", url)

		cd, err := chunked.New(ctx.Context, client, url, settings)
		if err != nil {
			// ranged requests not supported, fallback to sequential download
			err = downloadSequential(ctx, client, url, file, settings)
			if err != nil {
				lastErr = err
				continue
			}
		} else {
			err = cd.Download(ctx, file, settings.NumConnections)
			if err != nil {
				lastErr = err
				continue
			}
		}

		outputPath := strings.TrimSuffix(
			filePath,
			filepath.Ext(filePath),
		) + "_remuxed" + filepath.Ext(filePath)
		ctx.FilesTracker.Add(outputPath)

		err = libav.RemuxFile(filePath, outputPath)
		if err != nil {
			ctx.Warnf("remuxing failed, using original file: %v", err)
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
	ctx.FilesTracker.Add(filePath)

	tempDir := ToPath("segments" + uuid.NewString()[:8])
	ctx.FilesTracker.Add(tempDir)

	if err := os.MkdirAll(tempDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create temporary directory: %w", err)
	}

	ctx.Debugf("attempting download from: %s", segmentURLs[0])

	sd := segmented.New(
		ctx.Context, client,
		tempDir, segmentURLs,
		&segmented.SegmentedDownloaderOptions{
			InitSegment:      initSegmentURL,
			DownloadSettings: settings,
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
	ctx.FilesTracker.Add(outputPath)

	err = libav.RemuxFile(filePath, outputPath)
	if err != nil {
		ctx.Warnf("remuxing failed, using original file: %v", err)
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
	settings = ensureDownloadSettings(settings)

	client := ctx.HTTPClient.AsDownloadClient()
	maxRetries := max(settings.Retries, 1)

	for _, url := range urlList {
		for attempt := range maxRetries {
			ctx.Debugf("attempting download from: %s (attempt %d/%d)", url, attempt+1, maxRetries)
			resp, err := client.FetchWithContext(
				ctx.Context,
				http.MethodGet,
				url, &networking.RequestParams{
					Headers: settings.Headers,
					Cookies: settings.Cookies,
				},
			)
			if err != nil {
				continue
			}

			if resp.StatusCode != http.StatusOK {
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

func downloadSequential(
	ctx *models.ExtractorContext,
	client *networking.HTTPClient,
	url string,
	writer io.Writer,
	settings *models.DownloadSettings,
) error {
	maxRetries := max(settings.Retries, 1)

	for attempt := range maxRetries {
		ctx.Debugf("sequential download attempt %d/%d", attempt+1, maxRetries)

		resp, err := client.FetchWithContext(
			ctx.Context,
			http.MethodGet,
			url,
			&networking.RequestParams{
				Headers: settings.Headers,
				Cookies: settings.Cookies,
			},
		)
		if err != nil {
			ctx.Debugf("download attempt %d failed: %v", attempt+1, err)
			continue
		}

		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			logger.L.Debugf("download attempt %d got status %d", attempt+1, resp.StatusCode)
			continue
		}

		_, err = io.Copy(writer, resp.Body)
		if err != nil {
			resp.Body.Close()
			logger.L.Debugf("download attempt %d copy failed: %v", attempt+1, err)
			continue
		}

		resp.Body.Close()
		return nil
	}

	return fmt.Errorf("all sequential download attempts failed")
}
