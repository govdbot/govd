package networking

import (
	"net/http"
	"net/url"
	"time"

	"github.com/govdbot/govd/internal/logger"
)

var defaultTimeout = 30 * time.Second

func NewHTTPClient(options *NewHTTPClientOptions) *HTTPClient {
	if options == nil {
		options = &NewHTTPClientOptions{}
	}
	client := DefaultHTTPClient(options)

	if options.Proxy != "" {
		proxyURL, err := url.Parse(options.Proxy)
		if err != nil {
			logger.L.Warnf("invalid proxy URL: %v", err)
		} else {
			client.Client = &http.Client{
				Transport: NewTransportWithProxy(proxyURL),
				Timeout:   defaultTimeout,
			}
			client.Proxy = options.Proxy
		}
	} else if options.EdgeProxy != "" {
		client.Client = NewEdgeProxyClient(options.EdgeProxy)
		client.EdgeProxy = options.EdgeProxy
	} else if options.DisableProxy {
		client.Client = &http.Client{
			Transport: NewTransportNoProxyFromEnv(),
			Timeout:   defaultTimeout,
		}
		client.DisableProxy = true
	}
	if options.Impersonate {
		client.Client = NewChromeClient()
	}

	client.DownloadProxy = options.DownloadProxy
	return client
}

func DefaultHTTPClient(options *NewHTTPClientOptions) *HTTPClient {
	if options == nil {
		options = &NewHTTPClientOptions{}
	}
	return &HTTPClient{
		Client: &http.Client{
			Transport: NewTransport(),
			Timeout:   defaultTimeout,
		},
		Headers: options.Headers,
		Cookies: options.Cookies,
	}
}

func (c *HTTPClient) AsDownloadClient() *HTTPClient {
	client := DefaultHTTPClient(&NewHTTPClientOptions{
		Headers: c.Headers,
		Cookies: c.Cookies,
	})
	if c.DownloadProxy != "" {
		proxyURL, err := url.Parse(c.DownloadProxy)
		if err != nil {
			logger.L.Warnf("invalid download proxy URL: %v", err)
			return c
		}
		client.Client = &http.Client{
			Transport: NewTransportWithProxy(proxyURL),
			Timeout:   defaultTimeout,
		}
		client.DownloadProxy = c.DownloadProxy
	} else if c.DisableProxy {
		client.Client = &http.Client{
			Transport: NewTransportNoProxyFromEnv(),
			Timeout:   defaultTimeout,
		}
		client.DisableProxy = true
	}
	return client
}
