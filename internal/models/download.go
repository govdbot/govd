package models

import "net/http"

type DownloadSettings struct {
	NumConnections int
	ChunkSize      int64
	Headers        map[string]string
	Cookies        []*http.Cookie
	DecryptionKey  *DecryptionKey
	Retries        int
}
