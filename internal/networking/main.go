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

	switch {
	case options.Proxy != "":
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
	case options.EdgeProxy != "":
		client.Client = NewEdgeProxyClient(options.EdgeProxy)
		client.EdgeProxy = options.EdgeProxy
	case options.DisableProxy:
		client.Client = &http.Client{
			Transport: NewTransportNoProxyFromEnv(),
			Timeout:   defaultTimeout,
		}
		client.DisableProxy = true
	}
	if options.Impersonate {
		// Preserve proxy setting when impersonating
		var proxyURL *url.URL
		var useEdge, disableProxy bool
		var edgeURL string
		if options.Proxy != "" {
			if u, err := url.Parse(options.Proxy); err == nil {
				proxyURL = u
			}
		}
		if options.EdgeProxy != "" {
			useEdge = true
			edgeURL = options.EdgeProxy
		}
		if options.DisableProxy {
			disableProxy = true
		}
		chromeClient := NewChromeClient()
		// apply proxy onto chrome transport
		if t, ok := chromeClient.Transport.(*http.Transport); ok {
			if proxyURL != nil {
				t.Proxy = http.ProxyURL(proxyURL)
			} else if useEdge {
				// edge proxy client already handles it, keep chrome for non-edge
			} else if disableProxy {
				t.Proxy = func(_ *http.Request) (*url.URL, error) { return nil, nil }
			}
		}
		if useEdge && options.Proxy == "" {
			// edge proxy case: keep edge client but with chrome TLS would need custom
			// for simplicity, use chrome client (edge proxy is not compatible with chrome impersonate currently)
			// log warning
			logger.L.Debugf("impersonate + edge proxy both set, using impersonate")
		}
		client.Client = chromeClient
		// restore proxy fields for AsDownloadClient to use
		if proxyURL != nil {
			client.Proxy = options.Proxy
		}
		if useEdge {
			client.EdgeProxy = edgeURL
		}
		client.DisableProxy = disableProxy
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
