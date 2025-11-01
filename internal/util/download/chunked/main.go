package chunked

import (
	"context"
	"fmt"
	"io"
	"maps"
	"net/http"
	"sync"

	"github.com/govdbot/govd/internal/models"
	"github.com/govdbot/govd/internal/networking"
)

type ChunkedDownloader struct {
	client    *networking.HTTPClient
	url       string
	totalSize int64
	numChunks int
	settings  *models.DownloadSettings

	wg sync.WaitGroup
}

type Chunk struct {
	index  int
	reader io.ReadCloser
	err    error
}

func NewChunkedDownloader(
	ctx context.Context,
	client *networking.HTTPClient,
	url string,
	settings *models.DownloadSettings,
) (*ChunkedDownloader, error) {
	if client == nil {
		return nil, fmt.Errorf("http client cannot be nil")
	}

	resp, err := client.FetchWithContext(
		ctx, http.MethodHead, url,
		&networking.RequestParams{
			Cookies: settings.Cookies,
		},
	)

	chunkSize := settings.ChunkSize

	if err == nil {
		defer resp.Body.Close()
		totalSize := resp.ContentLength
		if totalSize > 0 && resp.Header.Get("Accept-Ranges") == "bytes" {
			numChunks := int((totalSize + chunkSize - 1) / chunkSize)
			return &ChunkedDownloader{
				client:    client,
				url:       resp.Request.URL.String(),
				totalSize: totalSize,
				numChunks: numChunks,
				settings:  settings,
			}, nil
		}
	}

	headers := map[string]string{
		"Range": "bytes=0-0",
	}
	maps.Copy(headers, settings.Headers)

	resp, err = client.FetchWithContext(
		ctx, http.MethodGet,
		url, &networking.RequestParams{
			Headers: headers,
			Cookies: settings.Cookies,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to determine content length (head fallback): %w", err)
	}
	defer resp.Body.Close()

	if cr := resp.Header.Get("Content-Range"); cr != "" {
		total, parseErr := parseContentRange(cr)
		if parseErr == nil && total > 0 {
			if resp.StatusCode != http.StatusPartialContent {
				return nil, fmt.Errorf("expected 206 for ranged GET, got %d", resp.StatusCode)
			}
			numChunks := int((total + chunkSize - 1) / chunkSize)
			return &ChunkedDownloader{
				client:    client,
				url:       resp.Request.URL.String(),
				totalSize: total,
				numChunks: numChunks,
				settings:  settings,
			}, nil
		}
	}

	if resp.StatusCode == http.StatusOK && resp.ContentLength > 0 {
		if resp.Header.Get("Accept-Ranges") != "bytes" {
			return nil, fmt.Errorf("server does not support range requests")
		}
		totalSize := resp.ContentLength
		numChunks := int((totalSize + chunkSize - 1) / chunkSize)
		return &ChunkedDownloader{
			client:    client,
			url:       resp.Request.URL.String(),
			totalSize: totalSize,
			numChunks: numChunks,
			settings:  settings,
		}, nil
	}

	return nil, fmt.Errorf("content length not available or server does not support ranged requests")
}

func (cd *ChunkedDownloader) Download(
	ctx *models.ExtractorContext,
	writer io.Writer,
	maxConcurrency int,
) error {
	maxConcurrency = max(maxConcurrency, 1)

	chunks := make(chan *Chunk, cd.numChunks)
	semaphore := make(chan struct{}, maxConcurrency)

	cd.wg.Add(cd.numChunks)
	for i := 0; i < cd.numChunks; i++ {
		go func(index int) {
			defer cd.wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()
			cd.downloadChunk(ctx, index, chunks)
		}(i)
	}

	go func() {
		cd.wg.Wait()
		close(chunks)
	}()

	return cd.writeChunks(writer, chunks)
}

func (cd *ChunkedDownloader) downloadChunk(
	ctx *models.ExtractorContext,
	index int,
	chunks chan<- *Chunk,
) {
	chunkSize := cd.settings.ChunkSize

	start := int64(index) * chunkSize
	end := min(start+chunkSize-1, cd.totalSize-1)

	headers := map[string]string{
		"Range": fmt.Sprintf("bytes=%d-%d", start, end),
	}
	maps.Copy(headers, cd.settings.Headers)

	maxRetries := max(cd.settings.Retries, 1)
	var lastErr error

	for attempt := range maxRetries {
		resp, err := cd.client.FetchWithContext(
			ctx.Context,
			http.MethodGet,
			cd.url, &networking.RequestParams{
				Headers: headers,
				Cookies: cd.settings.Cookies,
			},
		)
		if err != nil {
			lastErr = fmt.Errorf("failed to download chunk %d (attempt %d/%d): %w", index, attempt+1, maxRetries, err)
			continue
		}

		if resp.StatusCode != http.StatusPartialContent {
			resp.Body.Close()
			lastErr = fmt.Errorf("expected status 206, got %d for chunk %d (attempt %d/%d)", resp.StatusCode, index, attempt+1, maxRetries)
			continue
		}

		chunks <- &Chunk{index: index, reader: resp.Body}
		return
	}

	chunks <- &Chunk{index: index, err: lastErr}
}

func (cd *ChunkedDownloader) writeChunks(writer io.Writer, chunks <-chan *Chunk) error {
	chunkWriter := newChunkWriter(writer, cd.numChunks)

	for chunk := range chunks {
		if err := chunkWriter.addChunk(chunk); err != nil {
			chunkWriter.cleanup()
			return err
		}
	}

	return chunkWriter.finalize()
}
