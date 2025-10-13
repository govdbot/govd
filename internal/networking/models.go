package networking

import (
	"io"
	"net/http"
)

type HTTPClientInterface interface {
	Do(req *http.Request) (*http.Response, error)
}

type HTTPClient struct {
	Client        HTTPClientInterface
	Headers       map[string]string
	Cookies       []*http.Cookie
	Proxy         string
	EdgeProxy     string
	DownloadProxy string
}

type NewHTTPClientOptions struct {
	Headers       map[string]string
	Cookies       []*http.Cookie
	Proxy         string
	EdgeProxy     string
	DownloadProxy string
	Impersonate   bool
}

type RequestParams struct {
	Body    io.Reader
	Headers map[string]string
	Cookies []*http.Cookie
}

type EdgeProxyResponse struct {
	URL        string            `json:"url"`
	StatusCode int               `json:"status_code"`
	Text       string            `json:"text"`
	Headers    map[string]string `json:"headers"`
	Cookies    []string          `json:"cookies"`
}
