package networking

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/govdbot/govd/models"

	"github.com/pkg/errors"
)

type EdgeProxyClient struct {
	client   *http.Client
	proxyURL string
}

func NewEdgeProxyClientFromConfig(cfg *models.ExtractorConfig) *EdgeProxyClient {
	var baseClient *http.Client
	if cfg.Impersonate {
		baseClient = NewChromeClient()
	} else {
		baseClient = &http.Client{
			Transport: GetBaseTransport(),
			Timeout:   60 * time.Second,
		}
	}
	return &EdgeProxyClient{
		client:   baseClient,
		proxyURL: cfg.EdgeProxyURL,
	}
}

func NewEdgeProxyClient(proxyURL string) *EdgeProxyClient {
	return &EdgeProxyClient{
		client: &http.Client{
			Transport: GetBaseTransport(),
			Timeout:   60 * time.Second,
		},
		proxyURL: proxyURL,
	}
}

func (c *EdgeProxyClient) Do(req *http.Request) (*http.Response, error) {
	if c.proxyURL == "" {
		return nil, errors.New("proxy URL is not set")
	}

	targetURL := req.URL.String()
	encodedURL := url.QueryEscape(targetURL)
	proxyURLWithParam := c.proxyURL + "?url=" + encodedURL

	bodyBytes, err := readRequestBody(req)
	if err != nil {
		return nil, err
	}

	proxyReq, err := http.NewRequest(
		req.Method,
		proxyURLWithParam,
		bytes.NewBuffer(bodyBytes),
	)
	if err != nil {
		return nil, fmt.Errorf("error creating proxy request: %w", err)
	}

	copyHeaders(req.Header, proxyReq.Header)

	proxyResp, err := c.client.Do(proxyReq)
	if err != nil {
		return nil, fmt.Errorf("proxy request failed: %w", err)
	}
	defer proxyResp.Body.Close()

	return parseProxyResponse(proxyResp, req)
}

func readRequestBody(req *http.Request) ([]byte, error) {
	if req.Body == nil {
		return nil, nil
	}

	bodyBytes, err := io.ReadAll(req.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading request body: %w", err)
	}

	req.Body.Close()
	req.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

	return bodyBytes, nil
}
