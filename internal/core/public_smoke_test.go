package core

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/govdbot/govd/internal/config"
	"github.com/govdbot/govd/internal/extractors/instagram"
	"github.com/govdbot/govd/internal/logger"
	"github.com/govdbot/govd/internal/models"
	"github.com/govdbot/govd/internal/networking"
	"go.uber.org/zap"
)

func TestLivePublicDownload(t *testing.T) {
	id := os.Getenv("GOVD_INSTAGRAM_LIVE_ID")
	if id == "" {
		t.Skip("explicit opt-in live download test")
	}
	logger.L = zap.NewNop().Sugar()
	config.Env.MaxFileSize = 50 * 1024 * 1024
	config.Env.MaxDuration = 5 * time.Minute
	config.Env.DownloadsDirectory = "/app/downloads"
	deadline, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	ctx := &models.ExtractorContext{Context: deadline, ContentID: id, ContentURL: "https://www.instagram.com/p/" + id + "/", Extractor: instagram.Extractor, HTTPClient: networking.DefaultHTTPClient(nil), FilesTracker: models.NewFilesTracker()}
	defer ctx.FilesTracker.Cleanup()
	media, err := instagram.GetYTDLPMedia(ctx)
	if err != nil {
		t.Fatal(err)
	}
	formats, err := downloadMediaFormats(ctx, media)
	if err != nil {
		t.Fatal(err)
	}
	if len(formats) != 1 {
		t.Fatalf("unexpected file count %d", len(formats))
	}
	file := formats[0]
	info, err := os.Stat(file.FilePath)
	if err != nil {
		t.Fatal(err)
	}
	if info.Size() <= 0 || info.Size() > config.Env.MaxFileSize || file.Format.Duration <= 0 {
		t.Fatal("invalid final media or limits")
	}
	t.Logf("FULL_DOWNLOAD_OK bytes=%d duration=%d width=%d height=%d", info.Size(), file.Format.Duration, file.Format.Width, file.Format.Height)
}
