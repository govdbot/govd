package facebook

import (
	"context"
	"errors"
	"github.com/govdbot/govd/internal/config"
	"github.com/govdbot/govd/internal/models"
	"github.com/govdbot/govd/internal/util"
	"strings"
	"testing"
	"time"
)

func TestPublicFacebookDestinations(t *testing.T) {
	for _, raw := range []string{"https://www.facebook.com/share/r/Abc123/", "https://m.facebook.com/watch/?v=1412825880920191", "https://www.facebook.com/reel/1412825880920191", "https://www.facebook.com/video.php?v=1412825880920191"} {
		if !publicFacebookURL(raw) {
			t.Errorf("rejected %s", raw)
		}
	}
	for _, raw := range []string{"https://facebook.com.evil.invalid/reel/123", "https://evil.invalid/https://facebook.com/reel/123", "https://user@facebook.com/reel/123", "https://facebook.com:8080/reel/123", "http://facebook.com/reel/123"} {
		if publicFacebookURL(raw) {
			t.Errorf("accepted %s", raw)
		}
	}
	for _, raw := range []string{"https://video.fbcdn.net.evil.invalid/a.mp4", "http://video.fbcdn.net/a.mp4", "https://x@video.fbcdn.net/a.mp4", "https://127.0.0.1/a.mp4"} {
		if publicFacebookMediaURL(raw) {
			t.Errorf("accepted media %s", raw)
		}
	}
}
func TestPublicFacebookMetadataBoundaries(t *testing.T) {
	old := config.Env
	defer func() { config.Env = old }()
	config.Env.MaxFileSize = 1024
	config.Env.MaxDuration = time.Minute
	ctx := &models.ExtractorContext{ContentID: "123456789", ContentURL: "https://www.facebook.com/reel/123456789", Extractor: Extractor, Context: context.Background()}
	good := publicVideo{ID: ctx.ContentID, URL: "https://video.fbcdn.net/a.mp4", Duration: 10, FileSize: 100}
	for _, tc := range []struct {
		name   string
		change func(*publicVideo)
		want   error
	}{
		{"valid", func(*publicVideo) {}, nil},
		{"wrong video", func(i *publicVideo) { i.ID = "987654321" }, util.ErrUnavailable},
		{"private destination", func(i *publicVideo) { i.URL = "https://127.0.0.1/a.mp4" }, util.ErrUnavailable},
		{"live", func(i *publicVideo) { i.IsLive = true }, util.ErrUnavailable},
		{"playlist", func(i *publicVideo) { i.Entries = []*publicVideo{&good, &good} }, util.ErrUnavailable},
		{"oversize", func(i *publicVideo) { i.FileSize = 1025 }, util.ErrFileTooLarge},
		{"too long", func(i *publicVideo) { i.Duration = 61 }, util.ErrDurationTooLong},
	} {
		t.Run(tc.name, func(t *testing.T) {
			i := good
			tc.change(&i)
			m, e := parsePublicVideo(ctx, &i)
			if tc.want == nil {
				if e != nil || m == nil {
					t.Fatal(e)
				}
			} else if !errors.Is(e, tc.want) {
				t.Fatalf("got %v", e)
			}
		})
	}
}
func TestPublicFacebookHelperOutputBound(t *testing.T) {
	b := &helperOutput{remaining: 8}
	if _, e := b.Write([]byte(strings.Repeat("x", 9))); e == nil || b.Len() != 0 {
		t.Fatal("unbounded helper output")
	}
}
func TestFacebookPostMayHaveDistinctVideoID(t *testing.T) {
	old := config.Env
	defer func() { config.Env = old }()
	config.Env.MaxDuration = time.Minute
	config.Env.MaxFileSize = 1024
	ctx := &models.ExtractorContext{ContentID: "27748111711544437", ContentURL: "https://m.facebook.com/61561662904469/posts/27748111711544437", Extractor: Extractor}
	info := &publicVideo{ID: "1412825880920191", URL: "https://video.fbcdn.net/a.mp4", Duration: 10, FileSize: 100}
	if _, e := parsePublicVideo(ctx, info); e != nil {
		t.Fatal("post and video IDs are separate namespaces", e)
	}
}
