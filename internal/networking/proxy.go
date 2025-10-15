package networking

import (
	"net/http"
	"net/url"
	"sync"

	"github.com/govdbot/govd/internal/config"
	"golang.org/x/net/http/httpproxy"
)

var (
	envProxyOnce      sync.Once
	envProxyFuncValue func(*url.URL) (*url.URL, error)
)

func proxyFromEnv(req *http.Request) (*url.URL, error) {
	return envProxyFunc()(req.URL)
}

func envProxyFunc() func(*url.URL) (*url.URL, error) {
	envProxyOnce.Do(func() {
		cfg := &httpproxy.Config{
			HTTPProxy:  config.Env.Proxy,
			HTTPSProxy: config.Env.Proxy,
		}
		envProxyFuncValue = cfg.ProxyFunc()
	})
	return envProxyFuncValue
}
