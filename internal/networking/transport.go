package networking

import (
	"net"
	"net/http"
	"net/url"
	"time"
)

func NewTransport() *http.Transport {
	return &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   defaultTimeout,
			KeepAlive: defaultTimeout,
		}).DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   5 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		MaxIdleConnsPerHost:   100,
		MaxConnsPerHost:       100,
		ResponseHeaderTimeout: 10 * time.Second,
		DisableCompression:    false,
	}
}

func NewTransportNoProxyFromEnv() *http.Transport {
	transport := NewTransport()
	transport.Proxy = func(_ *http.Request) (*url.URL, error) {
		return nil, nil
	}
	return transport
}

func NewTransportWithProxy(proxyURL *url.URL) *http.Transport {
	transport := NewTransport()
	transport.Proxy = http.ProxyURL(proxyURL)
	return transport
}
