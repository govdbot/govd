package networking

import (
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/govdbot/govd/config"
	"github.com/govdbot/govd/models"
)

var (
	defaultClient     *http.Client
	defaultClientOnce sync.Once
	extractorClients  = make(map[string]models.HTTPClient)
)

func GetDefaultHTTPClient() *http.Client {
	defaultClientOnce.Do(func() {
		defaultClient = &http.Client{
			Transport: GetBaseTransport(),
			Timeout:   60 * time.Second,
		}
	})
	return defaultClient
}

func GetBaseTransport() *http.Transport {
	return &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
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

func GetExtractorHTTPClient(extractor *models.Extractor) models.HTTPClient {
	if client, exists := extractorClients[extractor.CodeName]; exists {
		return client
	}

	cfg := config.GetExtractorConfig(extractor)
	if cfg == nil {
		return GetDefaultHTTPClient()
	}

	var client models.HTTPClient

	if cfg.EdgeProxyURL != "" {
		client = NewEdgeProxyClientFromConfig(cfg)
	} else {
		client = NewClientFromConfig(cfg)
	}
	extractorClients[extractor.CodeName] = client

	return client
}

func NewClientFromConfig(cfg *models.ExtractorConfig) *http.Client {
	var baseClient *http.Client
	if cfg.Impersonate {
		baseClient = NewChromeClient()
	} else {
		baseClient = GetDefaultHTTPClient()
	}
	transport := GetBaseTransport()
	if cfg.HTTPProxy != "" || cfg.HTTPSProxy != "" {
		configureProxyTransport(transport, cfg)
	}
	baseClient.Transport = transport
	return baseClient
}
