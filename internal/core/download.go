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
	format := item.GetDefaultFormat()
	if format == nil {
		formats <- &models.DownloadedFormat{
			Index: index,
			Error: fmt.Errorf("no default format found for media item at index %d", index),
		}
		return
	}

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
	formats <- downloadedFormat
}

func downloadFormat(
	ctx *models.ExtractorContext,
	index int,
	format *models.MediaFormat,
) (*models.DownloadedFormat, error) {
	if len(format.URL) == 0 {
		return nil, fmt.Errorf("no URL found for format %s", format.FormatID)
	}

	logger.L.Debugf("downloading media item with format %s", format.FormatID)

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
