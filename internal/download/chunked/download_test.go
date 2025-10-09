package chunked

import (
	"context"
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/govdbot/govd/internal/networking"
)

// TestDownload100MB performs a real download from Hetzner test file to ensure the
// chunked downloader writes data correctly and the final size matches.
//
// Note: This is an integration test that performs network IO and can take time.
// It is skipped by default unless the TEST_INTEGRATION env var is set.
func TestDownload100MB(t *testing.T) {
	if os.Getenv("TEST_INTEGRATION") == "" {
		t.Skip("skipping integration test; set TEST_INTEGRATION=1 to run")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	url := "https://ash-speed.hetzner.com/100MB.bin"
	chunkSize := int64(5 * 1024 * 1024) // 5MB
	maxConcurrency := 4

	client := networking.NewHTTPClient(nil)
	dl, err := NewChunkedDownloader(ctx, client, url, chunkSize)
	if err != nil {
		t.Fatalf("failed to create downloader: %v", err)
	}

	// create temp file
	f, err := ioutil.TempFile("", "chunked-test-*.bin")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer func() {
		f.Close()
		os.Remove(f.Name())
	}()

	if err := dl.Download(ctx, f, maxConcurrency); err != nil {
		t.Fatalf("download failed: %v", err)
	}

	// verify size roughly 100MB (allow small differences)
	st, err := f.Stat()
	if err != nil {
		t.Fatalf("stat failed: %v", err)
	}

	size := st.Size()
	if size < 99*1024*1024 || size > 101*1024*1024 {
		t.Fatalf("unexpected size: %d", size)
	}
}
