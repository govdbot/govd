package networking

import (
	"net/http"
	"net/url"

	"github.com/govdbot/govd/internal/config"
	"golang.org/x/net/http/httpproxy"
)

func proxyFromEnv(req *http.Request) (*url.URL, error) {
	cfg := &httpproxy.Config{
		HTTPProxy:  config.Env.Proxy,
		HTTPSProxy: config.Env.Proxy,
	}
	return cfg.ProxyFunc()(req.URL)
}
