package models

import (
	"os"

	"github.com/govdbot/govd/internal/logger"
)

type FilesTracker struct {
	Files []string
}

func NewFilesTracker() *FilesTracker {
	return &FilesTracker{
		Files: make([]string, 0),
	}
}

func (ft *FilesTracker) Add(files ...string) {
	ft.Files = append(ft.Files, files...)
}

func (ft *FilesTracker) Cleanup() {

	for _, fileName := range ft.Files {
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
	ft.Files = make([]string, 0)
}
