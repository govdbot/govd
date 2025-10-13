package models

type Plugin struct {
	ID      string
	RunFunc func(*ExtractorContext, *DownloadedFormat) error
}
