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
	for _, filePtr := range files {
		if filePtr != nil {
			ft.Files = append(ft.Files, filePtr)
		}
	}
}

func (ft *FilesTracker) Cleanup() {
	var seen map[string]bool = make(map[string]bool)

	for _, filePtr := range ft.Files {
		if filePtr == nil || *filePtr == "" || seen[*filePtr] {
			continue
		}
		seen[*filePtr] = true

		logger.L.Debugf("removing temporary file: %s", *filePtr)
		_ = os.Remove(*filePtr)
	}

	ft.Files = make([]*string, 0)
}
