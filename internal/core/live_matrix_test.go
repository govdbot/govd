package core

import (
	"encoding/json"
	"errors"
	"github.com/govdbot/govd/internal/config"
	"github.com/govdbot/govd/internal/database"
	"github.com/govdbot/govd/internal/extractors"
	"github.com/govdbot/govd/internal/logger"
	"github.com/govdbot/govd/internal/util"
	"go.uber.org/zap"
	"os"
	"testing"
	"time"
)

func TestLiveMatrix(t *testing.T) {
	raw := os.Getenv("GOVD_LIVE_MATRIX")
	if raw == "" {
		t.Skip("opt-in public download matrix")
	}
	var cases []struct{ Name, URL, Expected string }
	if err := json.Unmarshal([]byte(raw), &cases); err != nil {
		t.Fatal(err)
	}
	logger.L = zap.NewNop().Sugar()
	config.Env.Caching = false
	config.Env.MaxFileSize = 50 * 1024 * 1024
	config.Env.MaxDuration = 5 * time.Minute
	config.Env.DownloadsDirectory = "/app/downloads"
	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			started := time.Now()
			ctx := extractors.FromURL(c.URL)
			if ctx == nil {
				t.Fatal("supported input did not resolve")
			}
			defer ctx.CancelFunc()
			defer ctx.FilesTracker.Cleanup()
			ctx.SetChat(&database.GetOrCreateChatRow{ChatID: 101, Type: database.ChatTypePrivate, Language: "en", MediaAlbumLimit: 10, Nsfw: true})
			task, err := executeDownload(ctx, false)
			if c.Expected == "too_large" {
				if !errors.Is(err, util.ErrFileTooLarge) {
					t.Fatalf("wanted file limit, got %v", err)
				}
				t.Logf("EXPECTED_LIMIT seconds=%.2f", time.Since(started).Seconds())
				return
			}
			if c.Expected == "unavailable" {
				if err == nil {
					t.Fatal("unavailable fixture unexpectedly downloaded")
				}
				t.Logf("EXPECTED_UNAVAILABLE seconds=%.2f error=%v", time.Since(started).Seconds(), err)
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			var total int64
			for _, f := range task.Formats {
				info, e := os.Stat(f.FilePath)
				if e != nil {
					t.Fatal(e)
				}
				total += info.Size()
				if info.Size() <= 0 || info.Size() > config.Env.MaxFileSize {
					t.Fatal("invalid size")
				}
			}
			t.Logf("DOWNLOAD_OK items=%d bytes=%d seconds=%.2f", len(task.Formats), total, time.Since(started).Seconds())
			ctx.FilesTracker.Cleanup()
			entries, e := os.ReadDir(config.Env.DownloadsDirectory)
			if e != nil || len(entries) != 0 {
				t.Fatalf("temporary files left: %d %v", len(entries), e)
			}
		})
	}
}
