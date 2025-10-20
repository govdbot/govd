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
	index int
	data  []byte
	err   error
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

	// prefer HEAD, but some servers block/close HEAD requests (EOF). If HEAD fails
	// fallback to a small GET with Range to infer content length and range support.
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

	// fallback: try a ranged GET for the first byte to get Content-Range or Content-Length
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

	// prefer Content-Range: "bytes 0-0/12345"
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

	// fallback to Content-Length header on GET response. only accept this if
	// the response is a full GET (200). For a ranged GET the Content-Length
	// will be the size of the fragment (e.g. 1) and is not the total size.
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
			semaphore <- struct{}{}        // acquire
			defer func() { <-semaphore }() // release
			cd.downloadChunk(ctx, index, chunks)
		}(i)
	}

	// close chunks channel when all downloads complete
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

	resp, err := cd.client.FetchWithContext(
		ctx.Context,
		http.MethodGet,
		cd.url, &networking.RequestParams{
			Headers: headers,
			Cookies: cd.settings.Cookies,
		},
	)
	if err != nil {
		chunks <- &Chunk{index: index, err: fmt.Errorf("failed to download chunk %d: %w", index, err)}
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusPartialContent {
		chunks <- &Chunk{index: index, err: fmt.Errorf("expected status 206, got %d for chunk %d", resp.StatusCode, index)}
		return
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		chunks <- &Chunk{index: index, err: fmt.Errorf("failed to read chunk %d: %w", index, err)}
		return
	}

	chunks <- &Chunk{index: index, data: data}
}

func (cd *ChunkedDownloader) writeChunks(writer io.Writer, chunks <-chan *Chunk) error {
	nextIndex := 0
	chunkBuffer := make(map[int]*Chunk)
	chunksReceived := 0

	for chunk := range chunks {
		chunksReceived++

		if chunk.err != nil {
			return fmt.Errorf("chunk %d failed: %w", chunk.index, chunk.err)
		}

		chunkBuffer[chunk.index] = chunk

		for {
			if chunk, exists := chunkBuffer[nextIndex]; exists {
				if _, err := writer.Write(chunk.data); err != nil {
					return fmt.Errorf("failed to write chunk %d: %w", nextIndex, err)
				}
				delete(chunkBuffer, nextIndex)
				nextIndex++
			} else {
				break
			}
		}
	}

	if chunksReceived != cd.numChunks {
		return fmt.Errorf("expected %d chunks, got %d", cd.numChunks, chunksReceived)
	}

	if nextIndex != cd.numChunks {
		return fmt.Errorf("expected %d chunks, wrote %d", cd.numChunks, nextIndex)
	}

	return nil
}
