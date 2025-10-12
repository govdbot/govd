package models

import (
	"os"

	"github.com/govdbot/govd/internal/logger"
)

// utility to track files for cleanup
type FilesTracker struct {
	Files []*string
}

func NewFilesTracker() *FilesTracker {
	return &FilesTracker{
		Files: make([]*string, 0),
	}
}

func (ft *FilesTracker) Add(files ...*string) {
	ft.Files = append(ft.Files, files...)
}

func (ft *FilesTracker) Cleanup() {
	for _, file := range ft.Files {
		if file != nil && *file != "" {
			filePath := *file
			logger.L.Debugf("removing file: %s", filePath)
			_ = os.Remove(filePath)
		}
	}
	ft.Files = nil
}
