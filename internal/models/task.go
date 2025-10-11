package models

type TaskResult struct {
	Media    *Media
	Formats  []*DownloadedFormat
	IsStored bool
}
