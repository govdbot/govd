package networking

import (
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/govdbot/govd/internal/logger"
)

var defaultTimeout = 30 * time.Second

func NewHTTPClient(options *NewHTTPClientOptions) *HTTPClient {
	if options == nil {
		options = &NewHTTPClientOptions{}
	}
	client := DefaultHTTPClient(options)

	switch {
	case options.Proxy != "":
		proxyURL, err := url.Parse(options.Proxy)
		if err != nil {
			logger.L.Warnf("invalid proxy URL: %v", err)
		} else {
			client.Client = &http.Client{
				Transport:     NewTransportWithProxy(proxyURL),
				Timeout:       defaultTimeout,
				CheckRedirect: checkRedirect,
			}
			client.Proxy = options.Proxy
		}
	case options.EdgeProxy != "":
		client.Client = NewEdgeProxyClient(options.EdgeProxy)
		client.EdgeProxy = options.EdgeProxy
	case options.DisableProxy:
		client.Client = &http.Client{
			Transport:     NewTransportNoProxyFromEnv(),
			Timeout:       defaultTimeout,
			CheckRedirect: checkRedirect,
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
			Transport:     NewTransport(),
			Timeout:       defaultTimeout,
			CheckRedirect: checkRedirect,
		},
		Headers:       options.Headers,
		Cookies:       options.Cookies,
		ResponseLimit: 8 * 1024 * 1024,
	}
}

func (c *HTTPClient) AsDownloadClient() *HTTPClient {
	client := DefaultHTTPClient(&NewHTTPClientOptions{
		Headers: c.Headers,
		Cookies: c.Cookies,
	})
	client.ResponseLimit = 128 * 1024 * 1024
	client.Headers = make(map[string]string)
	for name, value := range c.Headers {
		if name != "" && !strings.EqualFold(name, cookieHeader) && !strings.EqualFold(name, authorizationHeader) && !strings.EqualFold(name, "Proxy-Authorization") {
			client.Headers[name] = value
		}
	}
	if c.DownloadProxy != "" {
		proxyURL, err := url.Parse(c.DownloadProxy)
		if err != nil {
			logger.L.Warnf("invalid download proxy URL: %v", err)
			return c
		}
		client.Client = &http.Client{
			Transport:     NewTransportWithProxy(proxyURL),
			Timeout:       defaultTimeout,
			CheckRedirect: checkRedirect,
		}
		client.DownloadProxy = c.DownloadProxy
	} else if c.DisableProxy {
		client.Client = &http.Client{
			Transport:     NewTransportNoProxyFromEnv(),
			Timeout:       defaultTimeout,
			CheckRedirect: checkRedirect,
		}
		client.DisableProxy = true
	}
	return client
}
