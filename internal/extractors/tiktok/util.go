package tiktok

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"

	"github.com/bytedance/sonic"
	"github.com/govdbot/govd/internal/models"
	"github.com/govdbot/govd/internal/networking"
	"github.com/govdbot/govd/internal/util"
)

const videoURLBase = "https://www.tiktok.com/@_/video/"

var (
	universalDataPattern = regexp.MustCompile(`(?s)<script[^>]+\bid="__UNIVERSAL_DATA_FOR_REHYDRATION__"[^>]*>(.*?)<\/script>`)
)

func GetVideoWeb(ctx *models.ExtractorContext) (*WebItemStruct, []*http.Cookie, error) {
	awemeID := ctx.ContentID
	if !videoIDPattern.MatchString(awemeID) {
		return nil, nil, util.ErrUnavailable
	}

	resp, err := ctx.Fetch(
		http.MethodGet,
		videoURLBase+awemeID+"?is_from_webapp=1&sender_device=mobile",
		&networking.RequestParams{
			Headers: publicTikTokHeaders(),
		},
	)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.Request.URL.Path == "/login" {
		return nil, nil, util.ErrAuthenticationNeeded
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read response body: %w", err)
	}

	itemStruct, err := ParseUniversalData(body)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse universal data: %w", err)
	}
	return itemStruct, resp.Cookies(), nil
}

func ParseUniversalData(body []byte) (*WebItemStruct, error) {
	matches := universalDataPattern.FindSubmatch(body)
	if len(matches) < 2 {
		return nil, fmt.Errorf("universal data not found")
	}

	var data struct {
		Scope map[string]json.RawMessage `json:"__DEFAULT_SCOPE__"`
	}
	if err := json.Unmarshal(matches[1], &data); err != nil {
		return nil, fmt.Errorf("invalid TikTok public data: %w", err)
	}
	var itemStructBytes json.RawMessage
	for _, key := range []string{"webapp.video-detail", "webapp.reflow.video.detail"} {
		var page struct {
			ItemInfo struct {
				Item json.RawMessage `json:"itemStruct"`
			} `json:"itemInfo"`
		}
		if raw := data.Scope[key]; len(raw) == 0 || json.Unmarshal(raw, &page) != nil {
			continue
		}
		if candidate := page.ItemInfo.Item; len(candidate) > 0 && string(candidate) != "null" {
			itemStructBytes = candidate
			break
		}
	}
	if len(itemStructBytes) == 0 {
		return nil, util.ErrUnavailable
	}

	var webItem WebItemStruct
	err := sonic.ConfigFastest.Unmarshal(itemStructBytes, &webItem)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal item struct: %w", err)
	}
	return &webItem, nil
}
