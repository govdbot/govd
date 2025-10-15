package core

import (
	"fmt"
	"sync"

	"github.com/govdbot/govd/internal/database"
	"github.com/govdbot/govd/internal/logger"
	"github.com/govdbot/govd/internal/models"
	"github.com/govdbot/govd/internal/util"
	"github.com/govdbot/govd/internal/util/download"
)

func downloadMediaFormats(
	ctx *models.ExtractorContext,
	media *models.Media,
) ([]*models.DownloadedFormat, error) {
	var wg sync.WaitGroup

	ctx.DownloadFunc = downloadFormat

	numItems := len(media.Items)
	formats := make(chan *models.DownloadedFormat, numItems)
	semaphore := make(chan struct{}, 3)

	wg.Add(numItems)
	for i := range numItems {
		go func(index int) {
			defer wg.Done()
			semaphore <- struct{}{}        // acquire
			defer func() { <-semaphore }() // release
			downloadItem(ctx, formats, media.Items[index], index)
		}(i)
	}

	// close chunks channel when all downloads complete
	go func() {
		wg.Wait()
		close(formats)
	}()

	return collectDownloadedFormats(formats, numItems)
}

func downloadItem(
	ctx *models.ExtractorContext,
	formats chan<- *models.DownloadedFormat,
	item *models.MediaItem,
	index int,
) {
	var format *models.MediaFormat
	if len(item.Formats) > 1 {
		format = item.GetDefaultFormat()
	} else {
		format = item.Formats[0]
	}

	if format == nil {
		formats <- &models.DownloadedFormat{
			Index: index,
			Error: fmt.Errorf("no default format found for media item at index %d", index),
		}
		return
	}

	logger.L.Debugf("selected format: %s", format.ToString())

	// validate format before download
	// to avoid downloading large files
	// or unsupported formats
	err := validateFormat(format)
	if err != nil {
		formats <- &models.DownloadedFormat{
			Index: index,
			Error: err,
		}
		return
	}

	downloadedFormat, err := downloadFormat(ctx, index, format)
	if err != nil {
		formats <- &models.DownloadedFormat{
			Index: index,
			Error: err,
		}
		return
	}

	// validate format again after download
	// in case metadata extraction is done
	// after download
	err = validateFormat(format)
	if err != nil {
		formats <- &models.DownloadedFormat{
			Index: index,
			Error: err,
		}
		return
	}

	// merge audio into video if needed
	mergeFormats(item, downloadedFormat)

	for _, plugin := range format.Plugins {
		if plugin != nil {
			logger.L.Debugf("running plugin: %s", plugin.ID)
			err := plugin.RunFunc(ctx, item, downloadedFormat)
			if err != nil {
				formats <- &models.DownloadedFormat{
					Index: index,
					Error: fmt.Errorf("plugin %s failed: %w", plugin.ID, err),
				}
				return
			}
		}
	}

	formats <- downloadedFormat
}

func downloadFormat(
	ctx *models.ExtractorContext,
	index int,
	format *models.MediaFormat,
) (*models.DownloadedFormat, error) {
	if len(format.URL) == 0 {
		return nil, fmt.Errorf("no URL found for selected format")
	}

	fileName := format.GetFileName()
	var filePath string
	var thumbnailFilePath string

	// track files for cleanup
	ctx.FilesTracker.Add(&filePath)
	ctx.FilesTracker.Add(&thumbnailFilePath)

	// for images, download in memory and convert to jpeg
	if format.Type == database.MediaTypePhoto {
		file, err := download.DownloadFileInMemory(ctx, format.URL)
		if err != nil {
			return nil, fmt.Errorf("failed to download image: %w", err)
		}

		filePath = download.ToPath(fileName)

		if err := util.ImgToJPEG(file, filePath, 0); err != nil {
			return nil, fmt.Errorf("failed to convert image: %w", err)
		}

		return &models.DownloadedFormat{
			Format:   format,
			Index:    index,
			FilePath: filePath,
		}, nil
	}

	// for video and audio, download to file
	filePath, err := download.DownloadFile(
		ctx, format.URL,
		fileName, format.DownloadSettings,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to download file: %w", err)
	}

	thumbnailFilePath, err = getThumbnail(ctx, format, filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to get thumbnail: %w", err)
	}

	if format.MissingMetadata() {
		// extract video metadata if missing
		// width, height, duration
		// this is needed for Telegram video messages
		// and for validating the format
		insertVideoInfo(format, filePath)
	}

	return &models.DownloadedFormat{
		Format:            format,
		Index:             index,
		FilePath:          filePath,
		ThumbnailFilePath: thumbnailFilePath,
	}, nil
}

func collectDownloadedFormats(
	formats chan *models.DownloadedFormat,
	numItems int,
) ([]*models.DownloadedFormat, error) {
	downloadedFormats := make([]*models.DownloadedFormat, numItems)

	var firstErr error
	formatsReceived := 0

	for df := range formats {
		formatsReceived++
		downloadedFormats[df.Index] = df
		if df.Error != nil && firstErr == nil {
			firstErr = df.Error
		}
		if formatsReceived == numItems {
			break
		}
	}

	return downloadedFormats, firstErr
}
