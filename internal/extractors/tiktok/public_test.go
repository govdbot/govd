package tiktok

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
)

func TestTikTokAnonymousCookieScopes(t *testing.T) {
	original := &http.Cookie{Name: "fixture", Value: "anonymous", Secure: true}
	result := publicTikTokCookies([]*http.Cookie{original, {Name: "site", Value: "anonymous", Domain: ".tiktok.com", Secure: true}, {Name: "foreign", Domain: ".evil.invalid"}, nil})
	if len(result) != 2 || result[0].Domain != "www.tiktok.com" || result[1].Domain != ".tiktok.com" || original.Domain != "" {
		t.Fatal("anonymous cookie scope widened or source mutated")
	}
}

const fixtureID = "7548343642138709278"

func embedFixture(id, url string, private bool) []byte {
	return []byte(fmt.Sprintf(`<script id="__FRONTITY_CONNECT_STATE__">{"source":{"data":{"/embed/v2/%s":{"isError":false,"videoData":{"itemInfos":{"id":"%s","text":"fixture","secret":%v,"video":{"urls":["%s"],"videoMeta":{"duration":17,"width":576,"height":1024}}}}}}}}</script>`, fixtureID, id, private, url))
}
func TestTikTokPublicEmbed(t *testing.T) {
	item, e := parseVideoEmbed(embedFixture(fixtureID, "https://v45.tiktokcdn-eu.com/a.mp4", false), fixtureID)
	if e != nil || item.Video == nil || item.Video.Duration != 17 || item.Video.Width != 576 || len(item.Video.PlayAddr.URLList) != 1 {
		t.Fatalf("item=%+v err=%v", item, e)
	}
	for _, body := range [][]byte{embedFixture("99999999999999", "https://v45.tiktokcdn-eu.com/a.mp4", false), embedFixture(fixtureID, "https://127.0.0.1/a.mp4", false), embedFixture(fixtureID, "https://cdn.tiktok.com.evil.invalid/a.mp4", false), embedFixture(fixtureID, "https://v45.tiktokcdn-eu.com/a.mp4", true), []byte(`<script id="__FRONTITY_CONNECT_STATE__">null</script>`)} {
		if _, e := parseVideoEmbed(body, fixtureID); e == nil {
			t.Fatal("accepted unavailable/foreign/private video")
		}
	}
}
func TestTikTokModernAndReflowSchemas(t *testing.T) {
	for _, scope := range []string{"webapp.video-detail", "webapp.reflow.video.detail"} {
		body := []byte(fmt.Sprintf(`<script id="__UNIVERSAL_DATA_FOR_REHYDRATION__">%s{"__DEFAULT_SCOPE__":{"%s":{"itemInfo":{"itemStruct":{"id":"%s","video":{"duration":17,"playAddr":"https://v16-webapp-prime.tiktok.com/a.mp4"}}}}}}</script>`, "\n", scope, fixtureID))
		item, e := ParseUniversalData(body)
		if e != nil || item.Video == nil || item.Video.PlayURL == "" || ensureItemIdentity(item, fixtureID) != nil {
			t.Fatal(e)
		}
		mixed := []byte(strings.Replace(string(body), `"__DEFAULT_SCOPE__":{`, `"__DEFAULT_SCOPE__":{"language":"it-IT","featureFlags":[true,false],"optional":null,`, 1))
		if item, e := ParseUniversalData(mixed); e != nil || item.ID != fixtureID {
			t.Fatalf("mixed unrelated scope data: %v", e)
		}
	}
	body := []byte(`<script id="__UNIVERSAL_DATA_FOR_REHYDRATION__">{"__DEFAULT_SCOPE__":{"recommendations":{"itemStruct":{"id":"` + fixtureID + `"}}}}</script>`)
	if _, e := ParseUniversalData(body); e == nil {
		t.Fatal("accepted recommendation as requested video")
	}
}
func TestTikTokEmbedMixedDataAndPrivacy(t *testing.T) {
	body := embedFixture(fixtureID, "https://v45.tiktokcdn-eu.com/a.mp4", false)
	mixed := []byte(strings.Replace(string(body), `"data":{`, `"data":{"language":"it-IT","optional":null,`, 1))
	if _, e := parseVideoEmbed(mixed, fixtureID); e != nil {
		t.Fatal(e)
	}
	if ensureItemIdentity(&WebItemStruct{ID: fixtureID, PrivateItem: true}, fixtureID) == nil {
		t.Fatal("private item accepted")
	}
}
func TestTikTokURLBoundaries(t *testing.T) {
	for _, raw := range []string{"https://www.tiktok.com/@/video/" + fixtureID, "https://www.tiktok.com/@_/video/" + fixtureID, "https://www.tiktok.com/@user/video/" + fixtureID, "https://www.tiktok.com/embed/v2/" + fixtureID, "https://www.tiktok.com/embed/" + fixtureID} {
		if idx := Extractor.URLPattern.FindStringIndex(raw); idx == nil || idx[0] != 0 {
			t.Fatal("unsupported official video URL", raw)
		}
	}
	for _, raw := range []string{"https://tiktok.com.evil.invalid/x", "https://user@tiktok.com/x", "https://tiktok.com:8080/x", "http://v45.tiktokcdn-eu.com/x"} {
		if publicTikTokURL(raw) {
			t.Fatal(raw)
		}
	}
	if ensureItemIdentity(nil, fixtureID) == nil {
		t.Fatal("nil item accepted")
	}
	if len(validTikTokURLs(strings.Fields(strings.Repeat("https://v45.tiktokcdn-eu.com/x ", 20)))) != 8 {
		t.Fatal("unbounded CDN list")
	}
}
