package models

import (
	"github.com/govdbot/govd/internal/logger"
	"go.uber.org/zap"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func TestFilesTrackerConcurrentCleanup(t *testing.T) {
	logger.L = zap.NewNop().Sugar()
	tracker := NewFilesTracker()
	dir := t.TempDir()
	var wg sync.WaitGroup
	for range 20 {
		f, err := os.CreateTemp(dir, "tracked-")
		if err != nil {
			t.Fatal(err)
		}
		f.Close()
		name := f.Name()
		wg.Add(1)
		go func() { defer wg.Done(); tracker.Add(name); tracker.Cleanup() }()
	}
	wg.Wait()
	tracker.Cleanup()
	files, err := filepath.Glob(filepath.Join(dir, "tracked-*"))
	if err != nil || len(files) != 0 {
		t.Fatalf("files=%v err=%v", files, err)
	}
}
func TestUploadHandlesClosedAndIdempotent(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "upload-")
	if err != nil {
		t.Fatal(err)
	}
	format := &MediaFormat{}
	format.uploadClosers = append(format.uploadClosers, f)
	format.CloseUploads()
	format.CloseUploads()
	if _, err = f.WriteString("must fail"); err == nil {
		t.Fatal("upload handle remains open")
	}
	if len(format.uploadClosers) != 0 {
		t.Fatal("retained handles")
	}
}
