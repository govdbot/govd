package models

import (
	"os"
)

// utility to track files for cleanup
type FilesTracker struct {
	Files map[*string]bool
}

func NewFilesTracker() *FilesTracker {
	return &FilesTracker{
		Files: make(map[*string]bool, 0),
	}
}

func (ft *FilesTracker) Add(files ...*string) {
	for _, filePtr := range files {
		if filePtr != nil && *filePtr != "" {
			ft.Files[filePtr] = true
		}
	}
}

func (ft *FilesTracker) Cleanup() {
	for filePtr := range ft.Files {
		if filePtr == nil || *filePtr == "" {
			continue
		}
		_ = os.Remove(*filePtr)
	}
	ft.Files = make(map[*string]bool, 0)
}
