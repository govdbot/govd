package tiktok

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"
)

func TestParseUniversalDataSupportsWebappVideoDetail(t *testing.T) {
	body := []byte(`<script id="__UNIVERSAL_DATA_FOR_REHYDRATION__">{"__DEFAULT_SCOPE__":{"webapp.video-detail":{"statusCode":0,"itemInfo":{"itemStruct":{"id":"123","desc":"caption","video":{"duration":12,"height":1280,"width":720,"PlayAddrStruct":{"Uri":"play","UrlList":["https://example.com/video.mp4"],"Width":720,"Height":1280}}}}}}}</script>`)

	item, status, err := ParseUniversalData(body)
	if err != nil {
		t.Fatalf("ParseUniversalData returned error: %v", err)
	}
	if status != 0 {
		t.Fatalf("expected status 0, got %d", status)
	}
	if item == nil || item.ID != "123" {
		t.Fatalf("expected parsed item 123, got %#v", item)
	}
	if item.Video == nil || item.Video.PlayAddr == nil || len(item.Video.PlayAddr.URLList) != 1 {
		t.Fatalf("expected parsed video play address, got %#v", item.Video)
	}
}

func TestSolveChallengeCookies(t *testing.T) {
	baseValue := []byte("seed")
	solution := "7"
	payload := append(append([]byte{}, baseValue...), []byte(solution)...)
	expectedDigest := sha256.Sum256(payload)

	challengePayload := `{"v":{"a":"` + base64.StdEncoding.EncodeToString(baseValue) + `","b":1777080052,"c":"` + base64.StdEncoding.EncodeToString(expectedDigest[:]) + `"},"s":"state"}`
	challengeEncoded := strings.TrimRight(base64.StdEncoding.EncodeToString([]byte(challengePayload)), "=")

	body := []byte(`<body>Please wait...<p id="wci" class="_wafchallengeid"></p><p id="cs" class="` + challengeEncoded + `"></p><p id="rci" class="waforiginalreid"></p><p id="rs" class="ref-value"></p></body>`)

	cookies, err := solveChallengeCookies(body)
	if err != nil {
		t.Fatalf("solveChallengeCookies returned error: %v", err)
	}
	if len(cookies) != 2 {
		t.Fatalf("expected 2 cookies, got %d", len(cookies))
	}

	var challengeValue string
	for _, cookie := range cookies {
		switch cookie.Name {
		case "_wafchallengeid":
			challengeValue = cookie.Value
		case "waforiginalreid":
			if cookie.Value != "ref-value" {
				t.Fatalf("expected referer cookie value ref-value, got %q", cookie.Value)
			}
		}
	}
	if challengeValue == "" {
		t.Fatal("expected _wafchallengeid cookie to be present")
	}

	decodedCookie, err := base64.StdEncoding.DecodeString(challengeValue)
	if err != nil {
		t.Fatalf("failed to decode challenge cookie value: %v", err)
	}

	var cookiePayload map[string]any
	if err := json.Unmarshal(decodedCookie, &cookiePayload); err != nil {
		t.Fatalf("failed to unmarshal challenge cookie payload: %v", err)
	}
	if cookiePayload["d"] != base64.StdEncoding.EncodeToString([]byte(solution)) {
		t.Fatalf("expected solved challenge payload, got %#v", cookiePayload["d"])
	}
}
