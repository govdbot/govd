package networking

import (
	"bytes"
	"fmt"
	"net/http"
	"net/url"

	"github.com/govdbot/govd/internal/logger"
)

type EdgeProxyClient struct {
	Client   HTTPClient
	ProxyURL string
}

func NewEdgeProxyClient(proxyURL string) HTTPClientInterface {
	return &EdgeProxyClient{
		Client: HTTPClient{
			Client: &http.Client{
				Timeout: defaultTimeout,
			},
		},
		ProxyURL: proxyURL,
	}
}

func (c *EdgeProxyClient) Do(req *http.Request) (*http.Response, error) {
	if c.ProxyURL == "" {
		return nil, fmt.Errorf("proxy URL is not set")
	}

	logger.L.Debug("routing request via edge proxy")

	targetURL := req.URL.String()
	encodedURL := url.QueryEscape(targetURL)
	proxyURLWithParam := c.ProxyURL + "?url=" + encodedURL

	body, err := readRequestBody(req)
	if err != nil {
		return nil, err
	}

	var headers = make(map[string]string)
	for key, values := range req.Header {
		if len(values) > 0 {
			headers[key] = values[0]
		}
	}
	resp, err := c.Client.Fetch(
		req.Method,
		proxyURLWithParam,
		&RequestParams{
			Body:    bytes.NewBuffer(body),
			Headers: headers,
			Cookies: req.Cookies(),
		},
	)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return parseEdgeProxyResponse(resp, req)
}
