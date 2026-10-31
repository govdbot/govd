package download

import (
	"context"
	"github.com/govdbot/govd/internal/config"
	"github.com/govdbot/govd/internal/logger"
	"github.com/govdbot/govd/internal/models"
	"github.com/govdbot/govd/internal/networking"
	"go.uber.org/zap"
	"io"
	"net/http"
	"os"
	"strings"
	"sync/atomic"
	"testing"
)

type downloadTransport func(*http.Request) (*http.Response, error)

func (f downloadTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func TestRejectedRangesRetryBoundedGETWithoutPartialPrefix(t *testing.T) {
	old := config.Env
	oldLogger := logger.L
	defer func() { config.Env = old; logger.L = oldLogger }()
	config.Env.MaxFileSize = 8
	config.Env.DownloadsDirectory = t.TempDir()
	logger.L = zap.NewNop().Sugar()
	var sequential atomic.Int32
	client := networking.DefaultHTTPClient(nil)
	client.Client = &http.Client{Transport: downloadTransport(func(r *http.Request) (*http.Response, error) {
		response := &http.Response{Request: r, Header: make(http.Header), StatusCode: http.StatusOK, ContentLength: 6, Body: io.NopCloser(strings.NewReader("abcdef"))}
		if r.Method == http.MethodHead {
			response.Header.Set("Accept-Ranges", "bytes")
			return response, nil
		}
		switch r.Header.Get("Range") {
		case "bytes=0-2":
			response.StatusCode = http.StatusPartialContent
			response.ContentLength = 3
			response.Body = io.NopCloser(strings.NewReader("abc"))
		case "bytes=3-5":
			response.StatusCode = http.StatusForbidden
			response.ContentLength = 0
			response.Body = io.NopCloser(strings.NewReader(""))
		case "":
			sequential.Add(1)
		default:
			t.Errorf("unexpected range %s", r.Header.Get("Range"))
		}
		return response, nil
	})}
	ctx := &models.ExtractorContext{Context: context.Background(), HTTPClient: client, FilesTracker: models.NewFilesTracker(), Extractor: &models.Extractor{ID: "fixture"}}
	defer ctx.FilesTracker.Cleanup()
	path, e := downloadFileWithClient(ctx, client, []string{"https://cdn.tiktok.com/fixture.mp4"}, "fixture.mp4", &models.DownloadSettings{ChunkSize: 3, NumConnections: 1, Retries: 1})
	if e != nil {
		t.Fatal(e)
	}
	content, e := os.ReadFile(path)
	if e != nil || string(content) != "abcdef" || sequential.Load() != 1 {
		t.Fatalf("content=%q sequential=%d err=%v", content, sequential.Load(), e)
	}
}
