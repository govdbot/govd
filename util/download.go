package util

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/govdbot/govd/config"
	"github.com/govdbot/govd/models"
	"github.com/govdbot/govd/util/libav"
	"github.com/govdbot/govd/util/networking"

	"github.com/dustin/go-humanize"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

var downloadHTTPClient = networking.GetDefaultHTTPClient()

func DownloadFile(
	ctx context.Context,
	urlList []string,
	fileName string,
	downloadConfig *models.DownloadConfig,
) (string, error) {
	zap.S().Debugf("invoking downloader: %v with config %+v", urlList, downloadConfig)

	var errs []error
	for _, fileURL := range urlList {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		default:
			if err := EnsureDownloadDir(); err != nil {
				return "", err
			}

			filePath := filepath.Join(config.Env.DownloadsDirectory, fileName)
			err := runChunkedDownload(ctx, fileURL, filePath, downloadConfig)
			if err != nil {
				errs = append(errs, err)
				continue
			}

			if downloadConfig.Remux {
				remuxedFilePath, err := libav.RemuxFile(filePath)
				if err != nil {
					os.Remove(filePath)
					return "", fmt.Errorf("remuxing failed: %w", err)
				}
				filePath = remuxedFilePath
			}
			return filePath, nil
		}
	}

	return "", fmt.Errorf("%v", errs)
}

func DownloadFileWithSegments(
	ctx context.Context,
	segmentURLs []string,
	fileName string,
	downloadConfig *models.DownloadConfig,
) (string, error) {
	zap.S().Debugf("invoking segments downloader: %s", fileName)

	if err := EnsureDownloadDir(); err != nil {
		return "", err
	}
	tempDir := filepath.Join(
		config.Env.DownloadsDirectory,
		"segments"+uuid.NewString(),
	)
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create temporary directory: %w", err)
	}

	downloadedFiles, err := downloadSegments(ctx, tempDir, segmentURLs, downloadConfig)
	if err != nil {
		os.RemoveAll(tempDir)
		return "", fmt.Errorf("failed to download segments: %w", err)
	}
	zap.S().Debugf("merging segments %d segments", len(downloadedFiles))
	mergedFilePath, err := libav.MergeSegments(downloadedFiles, fileName)
	if err != nil {
		os.RemoveAll(tempDir)
		return "", fmt.Errorf("failed to merge segments: %w", err)
	}
	if err := os.RemoveAll(tempDir); err != nil {
		return "", fmt.Errorf("failed to remove temporary directory: %w", err)
	}
	return mergedFilePath, nil
}

func DownloadFileInMemory(
	ctx context.Context,
	urlList []string,
	downloadConfig *models.DownloadConfig,
) (*bytes.Reader, error) {
	zap.S().Debugf("invoking in-memory downloader")

	var errs []error
	for _, fileURL := range urlList {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			data, err := downloadInMemory(
				ctx, fileURL,
				downloadConfig,
			)
			if err != nil {
				errs = append(errs, err)
				continue
			}
			return bytes.NewReader(data), nil
		}
	}

	return nil, fmt.Errorf("%v", errs)
}

func downloadInMemory(
	ctx context.Context,
	fileURL string,
	downloadConfig *models.DownloadConfig,
) ([]byte, error) {
	reqCtx, cancel := context.WithTimeout(
		ctx,
		downloadConfig.Timeout,
	)
	defer cancel()

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
		// continue with the request
	}

	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, fileURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	for key, value := range downloadConfig.Headers {
		req.Header.Set(key, value)
	}
	for _, cookie := range downloadConfig.Cookies {
		req.AddCookie(cookie)
	}

	resp, err := downloadHTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to download file: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	if resp.ContentLength > int64(downloadConfig.MaxInMemory) {
		return nil, fmt.Errorf("file too large for in-memory download: %s", humanize.Bytes(uint64(resp.ContentLength)))
	}

	// allocate a single buffer with the
	// correct size upfront to prevent reallocations
	var data []byte
	if resp.ContentLength > 0 {
		data = make([]byte, 0, resp.ContentLength)
	} else {
		// 64KB initial capacity
		data = make([]byte, 0, 64*1024)
	}

	// use a limited reader to prevent
	// exceeding memory limits even if content-length is wrong
	limitedReader := io.LimitReader(resp.Body, int64(downloadConfig.MaxInMemory))

	buf := make([]byte, 32*1024) // 32KB buffer
	for {
		n, err := limitedReader.Read(buf)
		if n > 0 {
			data = append(data, buf[:n]...)
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to read response body: %w", err)
		}
	}

	return data, nil
}

func EnsureDownloadDir() error {
	dir := config.Env.DownloadsDirectory
	if _, err := os.Stat(dir); err != nil {
		if os.IsNotExist(err) {
			zap.S().Debugf("creating downloads directory: %s", dir)
			if err := os.MkdirAll(dir, 0755); err != nil {
				return fmt.Errorf("failed to create downloads directory: %w", err)
			}
		} else {
			return fmt.Errorf("error accessing directory: %w", err)
		}
	}
	return nil
}

func runChunkedDownload(
	ctx context.Context,
	fileURL string,
	filePath string,
	downloadConfig *models.DownloadConfig,
) error {
	// reduce concurrency if it's greater
	// than the number of available CPUs
	maxProcs := runtime.GOMAXPROCS(0)
	optimalConcurrency := int(math.Max(1, float64(maxProcs-1)))

	if downloadConfig.Concurrency > optimalConcurrency {
		downloadConfig.Concurrency = optimalConcurrency
	}

	fileSize, err := getFileSize(ctx, fileURL, downloadConfig)
	if err != nil {
		return err
	}

	if ExceedsMaxFileSize(int64(fileSize)) {
		return ErrFileTooLarge
	}

	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	// pre-allocate file size if possible
	if fileSize > 0 {
		if err := file.Truncate(int64(fileSize)); err != nil {
			return fmt.Errorf("failed to allocate file space: %w", err)
		}
	}

	numChunks := 1
	if fileSize > 0 {
		numChunks = int(math.Ceil(float64(fileSize) / float64(downloadConfig.ChunkSize)))
	}

	semaphore := make(chan struct{}, downloadConfig.Concurrency)
	var wg sync.WaitGroup

	errChan := make(chan error, numChunks)
	var downloadErr error
	var errOnce sync.Once

	var completedChunks atomic.Int64
	var completedBytes atomic.Int64

	downloadCtx, cancelDownload := context.WithCancel(ctx)
	defer cancelDownload()

	// use a mutex to synchronize file access
	var fileMutex sync.Mutex

	for i := range numChunks {
		wg.Add(1)

		go func(chunkIndex int) {
			defer wg.Done()

			// calculate chunk bounds
			start := chunkIndex * downloadConfig.ChunkSize
			end := start + downloadConfig.ChunkSize - 1
			if end >= fileSize && fileSize > 0 {
				end = fileSize - 1
			}

			// respect concurrency limit
			select {
			case semaphore <- struct{}{}:
				defer func() { <-semaphore }()
			case <-downloadCtx.Done():
				return
			}

			err := downloadChunkToFile(
				downloadCtx, fileURL,
				file, start, end,
				downloadConfig, &fileMutex,
			)
			if err != nil {
				errOnce.Do(func() {
					downloadErr = fmt.Errorf("chunk %d: %w", chunkIndex, err)
					cancelDownload() // cancel all other downloads
					errChan <- downloadErr
				})
				return
			}

			// update progress
			chunkSize := end - start + 1
			completedChunks.Add(1)
			completedBytes.Add(int64(chunkSize))
			if fileSize > 0 {
				progress := float64(completedBytes.Load()) / float64(fileSize)
				if downloadConfig.ProgressUpdater != nil {
					downloadConfig.ProgressUpdater(progress)
				}
			}
		}(i)
	}

	done := make(chan struct{})

	go func() {
		wg.Wait()
		close(errChan)
		close(done)
	}()

	var multiErr []error

	select {
	case err := <-errChan:
		if err != nil {
			multiErr = append(multiErr, err)
			// collect all errors
			for e := range errChan {
				if e != nil {
					multiErr = append(multiErr, e)
				}
			}
		}
		<-done
	case <-ctx.Done():
		cancelDownload()
		<-done // wait for all goroutines to finish
		os.Remove(filePath)
		return ctx.Err()
	case <-done:
		// no errors
	}

	if len(multiErr) > 0 {
		os.Remove(filePath)
		return fmt.Errorf("multiple download errors: %v", multiErr)
	}

	return nil
}

func getFileSize(
	ctx context.Context,
	fileURL string,
	downloadConfig *models.DownloadConfig,
) (int, error) {
	size, err := getFileSizeWithHead(ctx, fileURL, downloadConfig)
	if err != nil {
		zap.S().Debugf("HEAD request failed: %v, trying fallback", err)
	} else if size > 0 {
		return size, nil
	}
	size, err = getFileSizeWithRange(ctx, fileURL, downloadConfig)
	if err != nil {
		return 0, fmt.Errorf("failed to get file size: %w", err)
	}
	if size == 0 {
		zap.S().Debugf("file size is unknown for URL: %s. using default chunk size", fileURL)
	}
	return size, err
}

func getFileSizeWithHead(
	ctx context.Context,
	fileURL string,
	downloadConfig *models.DownloadConfig,
) (int, error) {
	reqCtx, cancel := context.WithTimeout(ctx, downloadConfig.Timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodHead, fileURL, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to create HEAD request: %w", err)
	}

	for key, value := range downloadConfig.Headers {
		req.Header.Set(key, value)
	}
	for _, cookie := range downloadConfig.Cookies {
		req.AddCookie(cookie)
	}

	resp, err := downloadHTTPClient.Do(req)
	if err != nil {
		return 0, fmt.Errorf("failed to execute HEAD request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return 0, fmt.Errorf("HEAD request failed: status code %d", resp.StatusCode)
	}

	fileSize := int(resp.ContentLength)
	if fileSize > 0 {
		zap.S().Debugf("file size from HEAD: %s", humanize.Bytes(uint64(fileSize)))
		return fileSize, nil
	}

	// fallback to Content-Range header
	if contentRange := resp.Header.Get("Content-Range"); contentRange != "" {
		if parts := strings.Split(contentRange, "/"); len(parts) == 2 {
			if size, err := strconv.Atoi(parts[1]); err == nil && size > 0 {
				zap.S().Debugf("file size from Content-Range: %s", humanize.Bytes(uint64(size)))
				return size, nil
			}
		}
	}

	zap.S().Debug("HEAD request didn't return valid file size")
	return 0, nil
}

func getFileSizeWithRange(
	ctx context.Context,
	fileURL string,
	downloadConfig *models.DownloadConfig,
) (int, error) {
	reqCtx, cancel := context.WithTimeout(ctx, downloadConfig.Timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, fileURL, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to create request: %w", err)
	}

	for key, value := range downloadConfig.Headers {
		req.Header.Set(key, value)
	}
	for _, cookie := range downloadConfig.Cookies {
		req.AddCookie(cookie)
	}

	req.Header.Set("Range", "bytes=0-0")

	resp, err := downloadHTTPClient.Do(req)
	if err != nil {
		return 0, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if contentRange := resp.Header.Get("Content-Range"); contentRange != "" {
		// format is typically "bytes 0-0/1234" where 1234 is the total size
		parts := strings.Split(contentRange, "/")
		if len(parts) == 2 {
			size, err := strconv.Atoi(parts[1])
			if err == nil && size > 0 {
				zap.S().Debugf("file size from range: %s", humanize.Bytes(uint64(size)))
				return size, nil
			}
		}
	}

	return 0, nil
}

func downloadChunkToFile(
	ctx context.Context,
	fileURL string,
	file *os.File,
	start int,
	end int,
	downloadConfig *models.DownloadConfig,
	fileMutex *sync.Mutex,
) error {
	var lastErr error

	for attempt := 0; attempt <= downloadConfig.RetryAttempts; attempt++ {
		if attempt > 0 {
			// wait before retry
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(downloadConfig.RetryDelay):
			}
		}

		err := downloadAndWriteChunk(
			ctx, fileURL, file,
			start, end, downloadConfig,
			fileMutex,
		)
		if err == nil {
			return nil
		}

		zap.S().Debugf("chunk %d-%d download failed: %v", start, end, err)
		lastErr = err
	}

	return fmt.Errorf(
		"all %d attempts failed: %w",
		downloadConfig.RetryAttempts+1,
		lastErr,
	)
}

func downloadAndWriteChunk(
	ctx context.Context,
	fileURL string,
	file *os.File,
	start int,
	end int,
	downloadConfig *models.DownloadConfig,
	fileMutex *sync.Mutex,
) error {
	zap.S().Debugf(
		"downloading chunk %d-%d",
		start, end,
	)

	reqCtx, cancel := context.WithTimeout(ctx, downloadConfig.Timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, fileURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	for key, value := range downloadConfig.Headers {
		req.Header.Set(key, value)
	}
	for _, cookie := range downloadConfig.Cookies {
		req.AddCookie(cookie)
	}

	// set the range header for partial content
	req.Header.Add("Range", fmt.Sprintf("bytes=%d-%d", start, end))

	resp, err := downloadHTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("download failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusPartialContent && resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	chunkSize := resp.ContentLength
	zap.S().Debugf("chunk size: %s", humanize.Bytes(uint64(chunkSize)))

	// use a fixed-size buffer for
	// copying to avoid large allocations (32KB)
	buf := make([]byte, 32*1024)

	fileMutex.Lock()
	defer fileMutex.Unlock()

	if _, err := file.Seek(int64(start), io.SeekStart); err != nil {
		return fmt.Errorf("failed to seek file: %w", err)
	}

	_, err = io.CopyBuffer(file, resp.Body, buf)
	if err != nil {
		return fmt.Errorf("failed to write chunk data: %w", err)
	}

	return nil
}

func downloadFile(
	ctx context.Context,
	fileURL string,
	filePath string,
	downloadConfig *models.DownloadConfig,
) (string, error) {
	reqCtx, cancel := context.WithTimeout(ctx, downloadConfig.Timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, fileURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	for key, value := range downloadConfig.Headers {
		req.Header.Set(key, value)
	}
	for _, cookie := range downloadConfig.Cookies {
		req.AddCookie(cookie)
	}
	resp, err := downloadHTTPClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to download file: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	file, err := os.Create(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	// use a fixed-size buffer for
	// copying to avoid large allocations (32KB)
	buf := make([]byte, 32*1024)
	_, err = io.CopyBuffer(file, resp.Body, buf)
	if err != nil {
		return "", fmt.Errorf("failed to write file: %w", err)
	}

	return filePath, nil
}

func downloadSegments(
	ctx context.Context,
	path string,
	segmentURLs []string,
	downloadConfig *models.DownloadConfig,
) ([]string, error) {
	semaphore := make(chan struct{}, downloadConfig.Concurrency)
	var wg sync.WaitGroup

	var firstErr atomic.Value

	downloadedFiles := make([]string, len(segmentURLs))
	defer func() {
		if firstErr.Load() != nil {
			for _, path := range downloadedFiles {
				if path != "" {
					os.Remove(path)
				}
			}
		}
	}()

	downloadCtx, cancelDownload := context.WithCancel(ctx)
	defer cancelDownload()

	for i, segmentURL := range segmentURLs {
		wg.Add(1)
		go func(idx int, url string) {
			defer wg.Done()

			select {
			case <-downloadCtx.Done():
				return
			default:
				// continue with the download
			}

			// acquire semaphore slot
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			segmentFileName := fmt.Sprintf("segment_%05d", idx)
			segmentPath := filepath.Join(path, segmentFileName)

			filePath, err := downloadFile(
				ctx, url,
				segmentPath,
				downloadConfig,
			)

			if err != nil {
				if firstErr.Load() == nil {
					firstErr.Store(fmt.Errorf("failed to download segment %d: %w", idx, err))
					cancelDownload()
				}
				return
			}

			downloadedFiles[idx] = filePath
		}(i, segmentURL)
	}
	wg.Wait()

	if err := firstErr.Load(); err != nil {
		if e, ok := err.(error); ok {
			return nil, e
		}
		return nil, fmt.Errorf("unknown error: %v", err)
	}

	for i, file := range downloadedFiles {
		if file == "" {
			return nil, fmt.Errorf("segment %d was not downloaded", i)
		}
		if _, err := os.Stat(file); os.IsNotExist(err) {
			return nil, fmt.Errorf("segment %d file does not exist: %w", i, err)
		}
	}

	return downloadedFiles, nil
}
