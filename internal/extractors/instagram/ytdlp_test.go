package instagram

import (
	"context"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/govdbot/govd/internal/models"
	"github.com/govdbot/govd/internal/networking"
	"github.com/govdbot/govd/internal/util"
)

func testContext() *models.ExtractorContext {
	return &models.ExtractorContext{Context: context.Background(), ContentID: "B", ContentURL: "https://www.instagram.com/p/B/", Extractor: Extractor, HTTPClient: networking.DefaultHTTPClient(nil)}
}

func TestPublicCDNValidation(t *testing.T) {
	for _, value := range []string{"https://instagram.example.fna.fbcdn.net/video.mp4", "https://scontent.cdninstagram.com/video.mp4"} {
		if !publicInstagramCDN(value) {
			t.Errorf("rejected CDN %s", value)
		}
	}
	for _, value := range []string{"https://fbcdn.net.evil.test/video", "https://evilfbcdn.net/video", "https://127.0.0.1/video", "http://scontent.cdninstagram.com/video", "https://user:password@cdninstagram.com/video", "file:///etc/passwd", "https://evil.test/?u=https://instagram.com"} {
		if publicInstagramCDN(value) {
			t.Errorf("accepted untrusted URL %s", value)
		}
	}
}

func TestPublicHelperDoesNotReceiveCredentialsOrArbitraryURL(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX helper fixture")
	}
	dir := t.TempDir()
	script := `#!/bin/sh
if [ "${BOT_TOKEN+x}" = x ] || [ "${DB_PASSWORD+x}" = x ]; then exit 20; fi
case "$*" in *evil.example*) exit 21;; esac
case "$*" in *--ignore-config*--no-plugin-dirs*https://www.instagram.com/p/B/*) ;; *) exit 22;; esac
printf '%s' '{"url":"https://scontent.cdninstagram.com/video.mp4","duration":10,"filesize":1000}'
`
	if err := os.WriteFile(filepath.Join(dir, "yt-dlp"), []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)
	t.Setenv("BOT_TOKEN", "not-for-child")
	t.Setenv("DB_PASSWORD", "not-for-child")
	ctx := testContext()
	ctx.ContentURL = "https://evil.example/?u=https://www.instagram.com/p/B/"
	ctx.HTTPClient.Cookies = []*http.Cookie{{Name: "sessionid", Value: "personal", Domain: ".instagram.com"}}
	ctx.HTTPClient.Headers = map[string]string{"Cookie": "sessionid=personal", "Authorization": "personal"}
	media, err := GetYTDLPMedia(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(media.Items) != 1 {
		t.Fatal("missing video")
	}
	if len(ctx.HTTPClient.Cookies) != 0 || len(ctx.HTTPClient.Headers) != 0 {
		t.Fatal("download client retained credentials")
	}
}

func TestInvalidShortcodeRejectedBeforeStartingHelper(t *testing.T) {
	for _, id := range []string{"", "../evil", "B;command", strings.Repeat("B", 29)} {
		ctx := testContext()
		ctx.ContentID = id
		if _, err := GetYTDLPMedia(ctx); err == nil || !strings.Contains(err.Error(), "invalid Instagram shortcode") {
			t.Fatalf("invalid id %q: %v", id, err)
		}
	}
}

func TestHelperOutputBound(t *testing.T) {
	out := &cappedOutput{remaining: 3}
	if _, err := out.Write([]byte("four")); err == nil || out.Len() != 0 {
		t.Fatal("output cap was bypassed")
	}
}

func TestYTDLPMediaValidation(t *testing.T) {
	for _, info := range []*ytdlpMedia{nil, {URL: ""}, {URL: "https://evil.example/video.mp4"}, {Entries: []*ytdlpMedia{nil}}} {
		if _, err := parseYTDLPMedia(testContext(), info); err == nil {
			t.Fatal("invalid metadata accepted")
		}
	}
	media, err := parseYTDLPMedia(testContext(), &ytdlpMedia{Description: "caption", Entries: []*ytdlpMedia{{URL: "https://scontent.cdninstagram.com/one.mp4"}, {URL: "https://scontent.cdninstagram.com/two.mp4"}}})
	if err != nil || len(media.Items) != 2 || media.Caption != "caption" {
		t.Fatalf("album failed: %v", err)
	}
}

func TestIncompleteGQLVideo(t *testing.T) {
	for _, info := range []*Media{nil, {Typename: "GraphVideo"}, {Typename: "XDTGraphVideo", VideoURL: " "}} {
		if _, err := ParseGQLMedia(testContext(), info); !errors.Is(err, util.ErrUnavailable) {
			t.Fatalf("incomplete video: %v", err)
		}
	}
}

func TestCopyrightBlockedEmbed(t *testing.T) {
	fixture := `new ServerJS());s.handle({"contextJSON":"{\"context\":{\"copyright_blocked\":true},\"gql_data\":{\"shortcode_media\":{\"__typename\":\"GraphVideo\"}}}"});requireLazy`
	if _, err := ParseEmbedGQL([]byte(fixture)); !errors.Is(err, util.ErrUnavailable) {
		t.Fatalf("blocked embed: %v", err)
	}
}

func TestLivePublicMedia(t *testing.T) {
	id := os.Getenv("GOVD_INSTAGRAM_LIVE_ID")
	if id == "" {
		t.Skip("explicit opt-in live test")
	}
	ctx := testContext()
	ctx.ContentID = id
	media, err := GetYTDLPMedia(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(media.Items) == 0 {
		t.Fatal("empty media")
	}
	format := media.Items[0].Formats[0]
	resp, err := ctx.Fetch(http.MethodGet, format.URL[0], &networking.RequestParams{Headers: map[string]string{"Range": "bytes=0-1023"}})
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusPartialContent {
		t.Fatalf("CDN HTTP %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1024))
	if err != nil || !strings.Contains(string(body), "ftyp") {
		t.Fatal("not a readable MP4")
	}
	t.Logf("PUBLIC_OK items=%d bytes=%d type=%s", len(media.Items), len(body), resp.Header.Get(contentTypeHeader))
}
