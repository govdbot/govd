package models

func (format *MediaFormat) CloseUploads() {
	for _, file := range format.uploadClosers {
		file.Close()
	}
	format.uploadClosers = nil
}
