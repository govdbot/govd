package networking

import (
	"net/http"
	"time"
)

func NewHTTPClient(options *NewHTTPClientOptions) *HTTPClient {
	if options == nil {
		options = &NewHTTPClientOptions{}
	}
	return &HTTPClient{
		Client: &http.Client{
			Transport: NewTransport(),
			Timeout:   30 * time.Second,
		},
		Headers: options.Headers,
		Cookies: options.Cookies,
	}
}
