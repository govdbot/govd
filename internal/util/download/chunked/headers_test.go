package chunked

import (
	"context"
	"github.com/govdbot/govd/internal/models"
	"github.com/govdbot/govd/internal/networking"
	"io"
	"net/http"
	"strings"
	"testing"
)

type testTransport func(*http.Request) (*http.Response, error)

func (f testTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func TestHEADUsesDownloadHeadersAndRejectsErrorResponse(t *testing.T) {
	for _, code := range []int{http.StatusOK, http.StatusForbidden} {
		calls := 0
		client := networking.DefaultHTTPClient(nil)
		client.Client = &http.Client{Transport: testTransport(func(r *http.Request) (*http.Response, error) {
			calls++
			if r.Header.Get("User-Agent") != "fixture-agent" || r.Header.Get("Referer") != "https://www.tiktok.com/" {
				t.Error("download identity differs on HEAD and GET")
			}
			return &http.Response{Request: r, StatusCode: code, Header: http.Header{"Accept-Ranges": []string{"bytes"}}, ContentLength: 6, Body: io.NopCloser(strings.NewReader(""))}, nil
		})}
		settings := &models.DownloadSettings{ChunkSize: 3, Headers: map[string]string{"User-Agent": "fixture-agent", "Referer": "https://www.tiktok.com/"}}
		downloader, e := New(context.Background(), client, "https://cdn.tiktok.com/fixture.mp4", settings)
		if code == http.StatusOK {
			if e != nil || downloader == nil || calls != 1 {
				t.Fatal(e, calls)
			}
		} else if e == nil || downloader != nil || calls != 2 {
			t.Fatal("accepted an error page as a ranged media response", e, calls)
		}
	}
}
