package core

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"

	"github.com/govdbot/govd/config"
	"github.com/govdbot/govd/enums"
	"github.com/govdbot/govd/models"
	"github.com/govdbot/govd/util"

	"github.com/pkg/errors"
)

func DownloadMediaItem(
	ctx context.Context,
	media *models.Media,
	idx int,
) (*models.DownloadedMedia, error) {
	format := media.Format
	if format == nil {
		return nil, errors.New("media format is nil")
	}

	downloadConfig := models.GetDownloadConfig(format.DownloadConfig)

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

	if format.Type == enums.MediaTypePhoto {
		file, err := util.DownloadFileInMemory(ctx, format.URL, downloadConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to download image: %w", err)
		}
		path := filepath.Join(config.Env.DownloadsDirectory, fileName)
		if err := util.ImgToJPEG(file, path); err != nil {
			return nil, fmt.Errorf("failed to convert image: %w", err)
		}
		filePath = path
		cleanup = false
		return &models.DownloadedMedia{
			FilePath:          filePath,
			ThumbnailFilePath: thumbnailFilePath,
			Media:             media,
			Index:             idx,
		}, nil
	}

	// hndle non-photo (video/audio/other)
	if len(format.Segments) == 0 {
		path, err := util.DownloadFile(ctx, format.URL, fileName, downloadConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to download file: %w", err)
		}
		filePath = path
	} else {
		path, err := util.DownloadFileWithSegments(ctx, format.Segments, fileName, downloadConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to download segments: %w", err)
		}
		filePath = path
	}

	if format.Type == enums.MediaTypeVideo || format.Type == enums.MediaTypeAudio {
		path, err := GetFileThumbnail(ctx, format, filePath, downloadConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to get thumbnail: %w", err)
		}
		thumbnailFilePath = path
	}

	if format.Type == enums.MediaTypeVideo && (format.Width == 0 || format.Height == 0 || format.Duration == 0) {
		InsertVideoInfo(format, filePath)

		// check if the extracted video duration is too long
		if util.ExceedsMaxDuration(format.Duration) {
			return nil, util.ErrDurationTooLong
		}
	}

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
) (*models.DownloadedMedia, error) {
	return DownloadMediaItem(ctx, media, idx)
}

func StartConcurrentDownload(
	ctx context.Context,
	media *models.Media,
	resultsChan chan<- models.DownloadedMedia,
	errChan chan<- error,
	wg *sync.WaitGroup,
	idx int,
) {
	defer wg.Done()

	result, err := DownloadMediaItem(ctx, media, idx)
	if err != nil {
		errChan <- err
		return
	}

	resultsChan <- *result
}

func DownloadMedia(
	ctx context.Context,
	media *models.Media,
) (*models.DownloadedMedia, error) {
	return StartDownloadTask(ctx, media, 0)
}

func DownloadMedias(
	ctx context.Context,
	medias []*models.Media,
) ([]*models.DownloadedMedia, error) {
	if len(medias) == 0 {
		return []*models.DownloadedMedia{}, nil
	}

	if len(medias) == 1 {
		result, err := DownloadMedia(ctx, medias[0])
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
		go StartConcurrentDownload(
			ctx, media, resultsChan,
			errChan, &wg, idx,
		)
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
		for _, result := range results {
			if result.FilePath != "" {
				os.Remove(result.FilePath)
			}
			if result.ThumbnailFilePath != "" {
				os.Remove(result.ThumbnailFilePath)
			}
		}
		return nil, firstError
	}

	if len(results) > 1 {
		sort.SliceStable(results, func(i, j int) bool {
			return results[i].Index < results[j].Index
		})
	}

	return results, nil
}
