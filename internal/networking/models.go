package networking

import (
	"io"
	"net/http"
)

type HTTPClient struct {
	Client  *http.Client
	Headers map[string]string
	Cookies []*http.Cookie
}

type NewHTTPClientOptions struct {
	Headers map[string]string
	Cookies []*http.Cookie
}

type RequestParams struct {
	Body    io.Reader
	Headers map[string]string
	Cookies []*http.Cookie
}
