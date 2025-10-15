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
	for _, filePtr := range ft.Files {
		if filePtr == nil || *filePtr == "" {
			continue
		}
		fileName := *filePtr
		info, err := os.Stat(fileName)
		if err != nil {
			continue
		}
		if !info.IsDir() {
			err = os.Remove(*filePtr)
			if err == nil {
				logger.L.Debugf("removed temporary file: %s", *filePtr)
			}
		} else {
			err = os.RemoveAll(*filePtr)
			if err == nil {
				logger.L.Debugf("removed temporary directory: %s", *filePtr)
			}
		}
	}

	ft.Files = make([]*string, 0)
}
