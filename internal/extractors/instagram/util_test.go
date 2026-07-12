package instagram

import (
	"context"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/govdbot/govd/internal/database"
	"github.com/govdbot/govd/internal/logger"
	"github.com/govdbot/govd/internal/models"
	"github.com/govdbot/govd/internal/networking"
)

func TestMain(m *testing.M) {
	logger.Init()
	os.Exit(m.Run())
}

// fakeClient returns a fixed HTTP response built from a JSON fixture file,
// regardless of the request. This lets us exercise the REAL GetWebpageMedia
// parse/decision logic without hitting Instagram.
type fakeClient struct {
	body []byte
}

func (f *fakeClient) Do(req *http.Request) (*http.Response, error) {
	return &http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(strings.NewReader(string(f.body))),
		Header:     make(http.Header),
	}, nil
}

func newTestContext(fixture string) *models.ExtractorContext {
	body, err := os.ReadFile(fixture)
	if err != nil {
		panic(err)
	}
	ext := &models.Extractor{ID: "instagram"}
	ctx := &models.ExtractorContext{
		ContentURL:  "https://www.instagram.com/reel/DZv2ZyLsfzD/",
		ContentID:   "DZv2ZyLsfzD",
		Extractor:   ext,
		Context:     context.Background(),
		HTTPClient:  &networking.HTTPClient{Client: &fakeClient{body: body}},
	}
	return ctx
}

// TestReelVideoDetection proves the fix: a reel whose web_info response has
// is_video:null but a non-empty video_versions list MUST be parsed as a VIDEO,
// not a photo. Regression test for the "reel sent as thumbnail" bug.
func TestReelVideoDetection(t *testing.T) {
	ctx := newTestContext("/tmp/ig_now.json")

	media, err := GetWebpageMedia(ctx)
	if err != nil {
		t.Fatalf("GetWebpageMedia returned error: %v", err)
	}
	if media == nil || len(media.Items) == 0 {
		t.Fatalf("no media items returned")
	}

	item := media.Items[0]
	formats := item.Formats
	if len(formats) == 0 {
		t.Fatalf("no formats on media item")
	}

	gotVideo := false
	for _, f := range formats {
		if f.Type == database.MediaTypeVideo && len(f.URL) > 0 {
			gotVideo = true
			t.Logf("VIDEO format found, url=%s thumbnail=%v", f.URL[0], f.ThumbnailURL)
		}
	}
	if !gotVideo {
		t.Fatalf("EXPECTED video format (reel has video_versions), but got: %+v", formats)
	}
}
