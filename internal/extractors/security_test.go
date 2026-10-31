package extractors

import (
	"github.com/govdbot/govd/internal/logger"
	"go.uber.org/zap"
	"testing"
)

func TestInputURLBoundaries(t *testing.T) {
	logger.L = zap.NewNop().Sugar()
	for _, raw := range []string{"https://instagram.audit/https://instagram.com/p/B/", "https://instagram.com@evil.invalid/p/B/", "https://instagram.com.evil.invalid/p/B/", "https://instagram.com:1234/p/B/", "file:///etc/passwd", "https://127.0.0.1/p/B/"} {
		if InputSupported(raw) {
			t.Errorf("unsafe input accepted %s", raw)
		}
		if ctx := FromURL(raw); ctx != nil {
			ctx.CancelFunc()
			t.Errorf("unsafe context accepted %s", raw)
		}
	}
	for _, raw := range []string{"https://www.instagram.com/p/B/?igsh=fixture", "www.instagram.com/reel/B/", "instagram.com/p/B/"} {
		ctx := FromURL(raw)
		if ctx == nil {
			t.Errorf("valid input rejected %s", raw)
		} else {
			ctx.CancelFunc()
		}
	}
}
