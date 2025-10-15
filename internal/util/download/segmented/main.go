package segmented

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sync"

	"github.com/govdbot/govd/internal/models"
	"github.com/govdbot/govd/internal/networking"
)

type SegmentedDownloader struct {
	client        *networking.HTTPClient
	path          string
	initSegment   string
	segments      []string
	decryptionKey *models.DecryptionKey

	wg sync.WaitGroup
}

type SegmentedDownloaderOptions struct {
	InitSegment   string
	DecryptionKey *models.DecryptionKey
}

type Segment struct {
	index    int
	filePath string
	err      error
}

func NewSegmentedDownloader(
	ctx context.Context,
	client *networking.HTTPClient,
	path string,
	segments []string,
	options *SegmentedDownloaderOptions,
) *SegmentedDownloader {
	if options == nil {
		options = &SegmentedDownloaderOptions{}
	}

	return &SegmentedDownloader{
		client:        client,
		path:          path,
		initSegment:   options.InitSegment,
		segments:      segments,
		decryptionKey: options.DecryptionKey,
	}
}

func (sd *SegmentedDownloader) Download(
	ctx context.Context,
	writer io.Writer,
	maxConcurrency int,
) error {
	numSegments := len(sd.segments)

	maxConcurrency = max(maxConcurrency, 1)

	segmentsCh := make(chan *Segment, numSegments)
	semaphore := make(chan struct{}, maxConcurrency)

	sd.wg.Add(numSegments)
	for i := range numSegments {
		go func(index int) {
			defer sd.wg.Done()
			semaphore <- struct{}{}        // acquire
			defer func() { <-semaphore }() // release
			sd.downloadSegment(
				ctx, index,
				sd.segments[index],
				segmentsCh,
			)
		}(i)
	}

	// close chunks channel when all downloads complete
	go func() {
		sd.wg.Wait()
		close(segmentsCh)
	}()

	segments, err := sd.collectSegments(segmentsCh)
	if err != nil {
		return err
	}

	// download init segment if available
	if sd.initSegment != "" {
		initSegmentPath := filepath.Join(sd.path, "init_segment")
		err := sd.downloadSegmentToFile(ctx, sd.initSegment, initSegmentPath)
		if err != nil {
			return fmt.Errorf("failed to download init segment: %w", err)
		}
	}

	if sd.decryptionKey != nil {
		err := sd.decryptSegments(segments)
		if err != nil {
			return fmt.Errorf("failed to decrypt segments: %w", err)
		}
	}

	return writeSegments(writer, sd.initSegment, segments)
}

func (sd *SegmentedDownloader) collectSegments(segments <-chan *Segment) ([]string, error) {
	collected := make([]string, len(sd.segments))
	for seg := range segments {
		if seg.err != nil {
			return nil, fmt.Errorf("failed to download segment %d: %w", seg.index, seg.err)
		}
		collected[seg.index] = seg.filePath
	}
	return collected, nil
}

func (sd *SegmentedDownloader) downloadSegment(
	ctx context.Context,
	index int,
	url string,
	segments chan<- *Segment,
) {
	segmentFileName := fmt.Sprintf("segment_%05d", index)
	segmentFilePath := filepath.Join(sd.path, segmentFileName)

	err := sd.downloadSegmentToFile(ctx, url, segmentFilePath)
	if err != nil {
		segments <- &Segment{index: index, err: err}
		return
	}

	segments <- &Segment{index: index, filePath: segmentFilePath, err: nil}
}

func (sd *SegmentedDownloader) downloadSegmentToFile(
	ctx context.Context,
	url string,
	filePath string,
) error {
	resp, err := sd.client.FetchWithContext(
		ctx, http.MethodGet,
		url, nil,
	)
	if err != nil {
		return fmt.Errorf("failed to fetch segment %q: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to fetch segment %q: status %d", url, resp.StatusCode)
	}

	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	_, err = io.Copy(file, resp.Body)
	if err != nil {
		return fmt.Errorf("failed to write segment to file: %w", err)
	}

	return nil
}
