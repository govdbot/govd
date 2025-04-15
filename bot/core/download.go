package core

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"

	"govd/enums"
	"govd/models"
	"govd/util"
)

func downloadMediaItem(
	ctx context.Context,
	media *models.Media,
	config *models.DownloadConfig,
	idx int,
) (*models.DownloadedMedia, error) {
	if config == nil {
		config = util.DefaultConfig()
	}

	format := media.Format
	if format == nil {
		return nil, fmt.Errorf("media format is nil")
	}

	fileName := format.GetFileName()
	var filePath string
	var thumbnailFilePath string

	cleanup := true
	defer func() {
		if cleanup {
			if filePath != "" {
				os.Remove(filePath)
			}
			if thumbnailFilePath != "" {
				os.Remove(thumbnailFilePath)
			}
		}
	}()

	if format.Type != enums.MediaTypePhoto {
		if len(format.Segments) == 0 {
			path, err := util.DownloadFile(
				ctx, format.URL,
				fileName, config,
			)
			if err != nil {
				return nil, fmt.Errorf("failed to download file: %w", err)
			}
			filePath = path
		} else {
			path, err := util.DownloadFileWithSegments(
				ctx, format.Segments,
				fileName, config,
			)
			if err != nil {
				return nil, fmt.Errorf("failed to download segments: %w", err)
			}
			filePath = path
		}

		if format.Type == enums.MediaTypeVideo || format.Type == enums.MediaTypeAudio {
			path, err := getFileThumbnail(format, filePath)
			if err != nil {
				return nil, fmt.Errorf("failed to get thumbnail: %w", err)
			}
			thumbnailFilePath = path
		}

		if format.Type == enums.MediaTypeVideo {
			if format.Width == 0 || format.Height == 0 || format.Duration == 0 {
				insertVideoInfo(format, filePath)
			}
		}
	} else {
		file, err := util.DownloadFileInMemory(ctx, format.URL, config)
		if err != nil {
			return nil, fmt.Errorf("failed to download image: %w", err)
		}
		path := filepath.Join(config.DownloadDir, fileName)
		if err := util.ImgToJPEG(file, path); err != nil {
			return nil, fmt.Errorf("failed to convert image: %w", err)
		}
		filePath = path
	}

	// all good, no need to delete files
	cleanup = false
	return &models.DownloadedMedia{
		FilePath:          filePath,
		ThumbnailFilePath: thumbnailFilePath,
		Media:             media,
		Index:             idx,
	}, nil
}

func StartDownloadTask(
	ctx context.Context,
	media *models.Media,
	idx int,
	config *models.DownloadConfig,
) (*models.DownloadedMedia, error) {
	return downloadMediaItem(ctx, media, config, idx)
}

func StartConcurrentDownload(
	ctx context.Context,
	media *models.Media,
	resultsChan chan<- models.DownloadedMedia,
	config *models.DownloadConfig,
	errChan chan<- error,
	wg *sync.WaitGroup,
	idx int,
) {
	defer wg.Done()

	result, err := downloadMediaItem(ctx, media, config, idx)
	if err != nil {
		errChan <- err
		return
	}

	resultsChan <- *result
}

func DownloadMedia(
	ctx context.Context,
	media *models.Media,
	config *models.DownloadConfig,
) (*models.DownloadedMedia, error) {
	return StartDownloadTask(ctx, media, 0, config)
}

func DownloadMedias(
	ctx context.Context,
	medias []*models.Media,
	config *models.DownloadConfig,
) ([]*models.DownloadedMedia, error) {
	if len(medias) == 0 {
		return []*models.DownloadedMedia{}, nil
	}

	if len(medias) == 1 {
		result, err := DownloadMedia(ctx, medias[0], config)
		if err != nil {
			return nil, err
		}
		return []*models.DownloadedMedia{result}, nil
	}

	resultsChan := make(chan models.DownloadedMedia, len(medias))
	errChan := make(chan error, len(medias))
	var wg sync.WaitGroup

	for idx, media := range medias {
		wg.Add(1)
		go StartConcurrentDownload(ctx, media, resultsChan, config, errChan, &wg, idx)
	}

	go func() {
		wg.Wait()
		close(resultsChan)
		close(errChan)
	}()

	var results []*models.DownloadedMedia
	var firstError error
	received := 0
	for received < len(medias) {
		select {
		case result, ok := <-resultsChan:
			if ok {
				resultCopy := result
				results = append(results, &resultCopy)
				received++
			}
		case err, ok := <-errChan:
			if ok && firstError == nil {
				firstError = err
				received++
			}
		case <-ctx.Done():
			if firstError == nil {
				firstError = ctx.Err()
			}
			received++
		}
	}

	if firstError != nil {
		return results, firstError
	}

	if len(results) > 1 {
		sort.SliceStable(results, func(i, j int) bool {
			return results[i].Index < results[j].Index
		})
	}

	return results, nil
}
