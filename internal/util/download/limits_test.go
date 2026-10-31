package download

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/govdbot/govd/internal/config"
	"github.com/govdbot/govd/internal/util"
)

func TestDownloadLimitWithoutContentLength(t *testing.T) {
	old := config.Env.MaxFileSize
	config.Env.MaxFileSize = 8
	t.Cleanup(func() { config.Env.MaxFileSize = old })
	if data, err := readLimited(strings.NewReader("12345678")); err != nil || len(data) != 8 {
		t.Fatalf("exact limit: %v", err)
	}
	if _, err := readLimited(strings.NewReader("123456789")); !errors.Is(err, util.ErrFileTooLarge) {
		t.Fatalf("unknown-length oversized response accepted: %v", err)
	}
	var buf bytes.Buffer
	writer := limitedWriter(&buf)
	if _, err := writer.Write([]byte("12345")); err != nil {
		t.Fatal(err)
	}
	if _, err := writer.Write([]byte("6789")); !errors.Is(err, util.ErrFileTooLarge) {
		t.Fatalf("stream limit bypassed: %v", err)
	}
	if buf.Len() > 8 {
		t.Fatal("wrote over the limit")
	}
}
