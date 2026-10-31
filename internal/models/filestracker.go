package models

import (
	"os"
	"sync"

	"github.com/govdbot/govd/internal/logger"
)

type FilesTracker struct {
	mu    sync.Mutex
	Files []string
}

func NewFilesTracker() *FilesTracker {
	return &FilesTracker{
		Files: make([]string, 0),
	}
}

func (ft *FilesTracker) Add(files ...string) {
	ft.mu.Lock()
	defer ft.mu.Unlock()
	ft.Files = append(ft.Files, files...)
}

func (ft *FilesTracker) Cleanup() {
	ft.mu.Lock()
	files := ft.Files
	ft.Files = nil
	ft.mu.Unlock()
	for _, fileName := range files {
		info, err := os.Stat(fileName)
		if err != nil {
			continue
		}
		if !info.IsDir() {
			err = os.Remove(fileName)
			if err == nil {
				logger.L.Debugf("removed temporary file: %s", fileName)
			}
		} else {
			err = os.RemoveAll(fileName)
			if err == nil {
				logger.L.Debugf("removed temporary directory: %s", fileName)
			}
		}
	}
}
