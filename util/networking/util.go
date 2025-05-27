package networking

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/govdbot/govd/models"

	"github.com/bytedance/sonic"
	"go.uber.org/zap"
)

func configureProxyTransport(
	transport *http.Transport,
	cfg *models.ExtractorConfig,
) {
	var httpProxyURL, httpsProxyURL *url.URL
	var err error

	if cfg.HTTPProxy != "" {
		httpProxyURL, err = url.Parse(cfg.HTTPProxy)
		if err != nil {
			zap.S().Warnf("invalid HTTP proxy URL '%s': %v", cfg.HTTPProxy, err)
		}
	}
	if cfg.HTTPSProxy != "" {
		httpsProxyURL, err = url.Parse(cfg.HTTPSProxy)
		if err != nil {
			zap.S().Warnf("invalid HTTPS proxy URL '%s': %v", cfg.HTTPSProxy, err)
		}
	}
	if httpProxyURL == nil && httpsProxyURL == nil {
		return
	}
	noProxyList := parseNoProxyList(cfg.NoProxy)
	transport.Proxy = func(req *http.Request) (*url.URL, error) {
		if shouldBypassProxy(req.URL.Hostname(), noProxyList) {
			return nil, nil
		}
		if req.URL.Scheme == "https" && httpsProxyURL != nil {
			return httpsProxyURL, nil
		}
		if req.URL.Scheme == "http" && httpProxyURL != nil {
			return httpProxyURL, nil
		}
		if httpsProxyURL != nil {
			return httpsProxyURL, nil
		}
		return httpProxyURL, nil
	}
}

func parseNoProxyList(noProxy string) []string {
	if noProxy == "" {
		return nil
	}
	list := strings.Split(noProxy, ",")
	for i := range list {
		list[i] = strings.TrimSpace(list[i])
	}
	return list
}

func shouldBypassProxy(host string, noProxyList []string) bool {
	for _, p := range noProxyList {
		if p == "" {
			continue
		}
		if p == host || (strings.HasPrefix(p, ".") && strings.HasSuffix(host, p)) {
			return true
		}
	}
	return false
}

func copyHeaders(source, destination http.Header) {
	for name, values := range source {
		for _, value := range values {
			destination.Add(name, value)
		}
	}
}

func parseProxyResponse(proxyResp *http.Response, originalReq *http.Request) (*http.Response, error) {

	var response models.EdgeProxyResponse
	decoder := sonic.ConfigFastest.NewDecoder(proxyResp.Body)
	if err := decoder.Decode(&response); err != nil {
		return nil, fmt.Errorf("error parsing proxy response: %w", err)
	}

	resp := &http.Response{
		StatusCode: response.StatusCode,
		Status:     strconv.Itoa(response.StatusCode) + " " + http.StatusText(response.StatusCode),
		Body:       io.NopCloser(bytes.NewBufferString(response.Text)),
		Header:     make(http.Header),
		Request:    originalReq,
	}

	parsedResponseURL, err := url.Parse(response.URL)
	if err != nil {
		return nil, fmt.Errorf("error parsing response URL: %w", err)
	}
	resp.Request.URL = parsedResponseURL

	for name, value := range response.Headers {
		resp.Header.Set(name, value)
	}

	for _, cookie := range response.Cookies {
		resp.Header.Add("Set-Cookie", cookie)
	}

	return resp, nil
}
