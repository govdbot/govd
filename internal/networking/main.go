package networking

import (
	"net/http"
	"net/url"
	"time"

	"github.com/govdbot/govd/internal/logger"
)

func NewHTTPClient(options *NewHTTPClientOptions) *HTTPClient {
	if options == nil {
		options = &NewHTTPClientOptions{}
	}
	transport := NewTransport()

	if options.Proxy != "" {
		proxyURL, err := url.Parse(options.Proxy)
		if err != nil {
			logger.L.Warnf("invalid proxy URL: %v", err)
		} else {
			transport = NewTransportWithProxy(proxyURL)
			return &HTTPClient{
				Client: &http.Client{
					Transport: transport,
					Timeout:   30 * time.Second,
				},
				Headers: options.Headers,
				Cookies: options.Cookies,
				Proxy:   options.Proxy,
			}
		}
	} else if options.EdgeProxy != "" {
		return &HTTPClient{
			Client:    NewEdgeProxyClient(options.EdgeProxy),
			Headers:   options.Headers,
			Cookies:   options.Cookies,
			EdgeProxy: options.EdgeProxy,
		}
	} else if options.Impersonate {
		return &HTTPClient{
			Client:  NewChromeClient(),
			Headers: options.Headers,
			Cookies: options.Cookies,
		}
	}
	return &HTTPClient{
		Client: &http.Client{
			Transport: transport,
			Timeout:   30 * time.Second,
		},
		Headers: options.Headers,
		Cookies: options.Cookies,
	}
}

func (c *HTTPClient) AsDownloadClient() *HTTPClient {
	if c.EdgeProxy != "" {
		// EdgeProxy clients are not suitable for downloads
		return &HTTPClient{
			Client: &http.Client{
				Transport: NewTransport(),
				Timeout:   60 * time.Second,
			},
			Headers: c.Headers,
			Cookies: c.Cookies,
		}
	}
	return c
}
