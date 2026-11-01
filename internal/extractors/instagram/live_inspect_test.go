package instagram

import (
	"context"
	"github.com/govdbot/govd/internal/logger"
	"github.com/govdbot/govd/internal/models"
	"github.com/govdbot/govd/internal/networking"
	"go.uber.org/zap"
	"net/http"
	"net/url"
	"os"
	"testing"
)

func TestLiveInspectPost(t *testing.T) {
	id := os.Getenv("GOVD_INSPECT_ID")
	if id == "" {
		t.Skip("operator diagnostic")
	}
	logger.L = zap.NewNop().Sugar()
	ctx := &models.ExtractorContext{Context: context.Background(), ContentID: id, ContentURL: "https://www.instagram.com/p/" + id + "/", Extractor: Extractor, HTTPClient: networking.DefaultHTTPClient(nil)}
	data, err := GetGQLData(ctx)
	t.Logf("GQL err=%v", err)
	if data != nil && data.ShortcodeMedia != nil {
		root := data.ShortcodeMedia
		t.Logf("ROOT type=%q", root.Typename)
		if root.EdgeSidecarToChildren != nil {
			for i, edge := range root.EdgeSidecarToChildren.Edges {
				if edge == nil || edge.Node == nil {
					t.Logf("CHILD %d nil", i)
					continue
				}
				t.Logf("CHILD %d type=%q image=%t video=%t", i, edge.Node.Typename, edge.Node.DisplayURL != "", edge.Node.VideoURL != "")
			}
		}
	}
	media, err := GetEmbedMedia(ctx)
	t.Logf("EMBED err=%v", err)
	if media != nil {
		for i, item := range media.Items {
			t.Logf("EMBED_ITEM %d formats=%d", i, len(item.Formats))
		}
	}
}
func TestLiveInspectShare(t *testing.T) {
	raw := os.Getenv("GOVD_INSPECT_SHARE")
	if raw == "" {
		t.Skip("operator diagnostic")
	}
	c := networking.DefaultHTTPClient(nil)
	resp, err := c.Fetch("GET", raw, &networking.RequestParams{StopRedirect: func(*http.Request) bool { return true }})
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	target, e := url.Parse(resp.Header.Get("Location"))
	if e != nil {
		t.Fatal(e)
	}
	t.Logf("SHARE status=%d locationHost=%s path=%s", resp.StatusCode, target.Hostname(), target.Path)
	if next := target.Query().Get("next"); next != "" {
		u, _ := url.Parse(next)
		if u != nil {
			t.Logf("NEXT host=%s path=%s", u.Hostname(), u.Path)
		}
	}
}
