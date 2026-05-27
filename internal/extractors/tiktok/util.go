package tiktok

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/bytedance/sonic"
	"github.com/govdbot/govd/internal/logger"
	"github.com/govdbot/govd/internal/models"
	"github.com/govdbot/govd/internal/networking"
	"github.com/govdbot/govd/internal/util"
)

const videoURLBase = "https://www.tiktok.com/@_/video/"
const maxChallengeSolution = 1_000_000

var (
	universalDataPattern = regexp.MustCompile(`<script[^>]+\bid="__UNIVERSAL_DATA_FOR_REHYDRATION__"[^>]*>(.*?)<\/script>`)
	htmlIDClassPattern   = regexp.MustCompile(`(?is)<[^>]+\bid="([^"]+)"[^>]*\bclass="([^"]*)"`)

	webHeaders = map[string]string{
		"Host":            "www.tiktok.com",
		"Connection":      "keep-alive",
		"User-Agent":      "Mozilla/5.0",
		"Accept":          "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8",
		"Accept-Language": "en-us,en;q=0.5",
		"Sec-Fetch-Mode":  "navigate",
	}
)

func GetVideoWeb(ctx *models.ExtractorContext) (*WebItemStruct, []*http.Cookie, error) {
	page, err := fetchVideoWebpage(ctx, nil)
	if err != nil {
		return nil, nil, err
	}

	itemStruct, status, err := ParseUniversalData(page.body)
	if err == nil {
		if statusErr := getVideoWebStatusError(status); statusErr != nil {
			return nil, nil, statusErr
		}
		return itemStruct, page.cookies, nil
	}

	if !isChallengePage(page.body) {
		return nil, nil, fmt.Errorf("failed to parse universal data: %w", err)
	}

	ctx.Debugf("solving TikTok web challenge")

	challengeCookies, err := solveChallengeCookies(page.body)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to solve TikTok challenge: %w", err)
	}

	page, err = fetchVideoWebpage(ctx, mergeCookies(page.cookies, challengeCookies))
	if err != nil {
		return nil, nil, err
	}

	itemStruct, status, err = ParseUniversalData(page.body)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse universal data after challenge: %w", err)
	}
	if statusErr := getVideoWebStatusError(status); statusErr != nil {
		return nil, nil, statusErr
	}
	return itemStruct, mergeCookies(page.cookies, challengeCookies), nil
}

type webPageResult struct {
	body    []byte
	cookies []*http.Cookie
}

func fetchVideoWebpage(
	ctx *models.ExtractorContext,
	cookies []*http.Cookie,
) (*webPageResult, error) {
	resp, err := ctx.Fetch(
		http.MethodGet,
		videoURLBase+ctx.ContentID,
		&networking.RequestParams{
			Headers: webHeaders,
			Cookies: cookies,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.Request.URL.Path == "/login" {
		return nil, util.ErrAuthenticationNeeded
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}
	return &webPageResult{
		body:    body,
		cookies: mergeCookies(resp.Cookies(), cookies),
	}, nil
}

func ParseUniversalData(body []byte) (*WebItemStruct, int, error) {
	matches := universalDataPattern.FindSubmatch(body)
	if len(matches) < 2 {
		return nil, 0, fmt.Errorf("universal data not found")
	}

	var data any
	err := sonic.ConfigFastest.Unmarshal(matches[1], &data)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to unmarshal universal data: %w", err)
	}
	logger.WriteFile("tt_universal_data", data)

	scope := data
	defaultScope := util.TraverseJSON(data, "__DEFAULT_SCOPE__")
	if defaultScope != nil {
		scope = defaultScope
		logger.WriteFile("tt_default_scope", defaultScope)
	}

	status := intFromJSON(util.TraverseJSON(scope, []string{"webapp.video-detail", "statusCode"}))
	itemStruct := util.TraverseJSON(scope, []string{"webapp.video-detail", "itemInfo", "itemStruct"})
	if itemStruct == nil {
		itemStruct = util.TraverseJSON(scope, "itemStruct")
	}
	if itemStruct == nil {
		if status != 0 {
			return nil, status, nil
		}
		return nil, status, util.ErrUnavailable
	}
	logger.WriteFile("tt_item_struct", itemStruct)

	itemStructBytes, err := sonic.ConfigFastest.Marshal(itemStruct)
	if err != nil {
		return nil, status, fmt.Errorf("failed to marshal item struct: %w", err)
	}

	var webItem WebItemStruct
	err = sonic.ConfigFastest.Unmarshal(itemStructBytes, &webItem)
	if err != nil {
		return nil, status, fmt.Errorf("failed to unmarshal item struct: %w", err)
	}
	return &webItem, status, nil
}

func isChallengePage(body []byte) bool {
	return bytes.Contains(body, []byte("Please wait...")) &&
		bytes.Contains(body, []byte(`id="cs"`)) &&
		bytes.Contains(body, []byte(`id="wci"`))
}

func solveChallengeCookies(body []byte) ([]*http.Cookie, error) {
	challengeCookieName, err := extractElementClass(body, "wci")
	if err != nil {
		return nil, err
	}
	challengeEncoded, err := extractElementClass(body, "cs")
	if err != nil {
		return nil, err
	}

	challengeBytes, err := decodeTikTokBase64(challengeEncoded)
	if err != nil {
		return nil, fmt.Errorf("failed to decode challenge payload: %w", err)
	}

	var challengeData map[string]any
	if err := sonic.ConfigFastest.Unmarshal(challengeBytes, &challengeData); err != nil {
		return nil, fmt.Errorf("failed to unmarshal challenge payload: %w", err)
	}

	baseHashInput, ok := util.TraverseJSON(challengeData, []string{"v", "a"}).(string)
	if !ok || baseHashInput == "" {
		return nil, fmt.Errorf("challenge base hash not found")
	}
	expectedDigestValue, ok := util.TraverseJSON(challengeData, []string{"v", "c"}).(string)
	if !ok || expectedDigestValue == "" {
		return nil, fmt.Errorf("challenge expected digest not found")
	}

	baseValue, err := decodeTikTokBase64(baseHashInput)
	if err != nil {
		return nil, fmt.Errorf("failed to decode challenge base value: %w", err)
	}
	expectedDigest, err := decodeTikTokBase64(expectedDigestValue)
	if err != nil {
		return nil, fmt.Errorf("failed to decode challenge expected digest: %w", err)
	}

	solution, err := solveChallengeNumber(baseValue, expectedDigest)
	if err != nil {
		return nil, err
	}
	challengeData["d"] = base64.StdEncoding.EncodeToString([]byte(solution))

	challengeCookieJSON, err := sonic.ConfigFastest.Marshal(challengeData)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal challenge cookie payload: %w", err)
	}

	cookies := []*http.Cookie{{
		Name:   challengeCookieName,
		Value:  base64.StdEncoding.EncodeToString(challengeCookieJSON),
		Domain: ".tiktok.com",
		Path:   "/",
	}}

	refererCookieName, _ := extractElementClass(body, "rci")
	refererCookieValue, _ := extractElementClass(body, "rs")
	if refererCookieName != "" {
		cookies = append(cookies, &http.Cookie{
			Name:   refererCookieName,
			Value:  refererCookieValue,
			Domain: ".tiktok.com",
			Path:   "/",
		})
	}

	return cookies, nil
}

func solveChallengeNumber(baseValue []byte, expectedDigest []byte) (string, error) {
	buffer := make([]byte, len(baseValue), len(baseValue)+8)
	copy(buffer, baseValue)

	for i := 0; i <= maxChallengeSolution; i++ {
		candidate := strconv.AppendInt(buffer[:len(baseValue)], int64(i), 10)
		sum := sha256.Sum256(candidate)
		if bytes.Equal(sum[:], expectedDigest) {
			return strconv.Itoa(i), nil
		}
	}
	return "", fmt.Errorf("unable to solve TikTok challenge")
}

func decodeTikTokBase64(value string) ([]byte, error) {
	padding := (4 - len(value)%4) % 4
	return base64.StdEncoding.DecodeString(value + strings.Repeat("=", padding))
}

func extractElementClass(body []byte, elementID string) (string, error) {
	for _, match := range htmlIDClassPattern.FindAllSubmatch(body, -1) {
		if len(match) < 3 || string(match[1]) != elementID {
			continue
		}
		return string(match[2]), nil
	}
	return "", fmt.Errorf("challenge element %q not found", elementID)
}

func intFromJSON(value any) int {
	switch v := value.(type) {
	case float64:
		return int(v)
	case int:
		return v
	case int32:
		return int(v)
	case int64:
		return int(v)
	case string:
		n, err := strconv.Atoi(v)
		if err == nil {
			return n
		}
	}
	return 0
}

func getVideoWebStatusError(status int) error {
	switch status {
	case 0:
		return nil
	case 10204:
		return util.ErrTikTokIPBlocked
	case 10216, 10222:
		return util.ErrAuthenticationNeeded
	default:
		return util.ErrUnavailable
	}
}

func mergeCookies(groups ...[]*http.Cookie) []*http.Cookie {
	merged := make([]*http.Cookie, 0)
	indexByKey := make(map[string]int)

	for _, group := range groups {
		for _, cookie := range group {
			if cookie == nil || cookie.Name == "" {
				continue
			}
			key := cookie.Name
			if index, ok := indexByKey[key]; ok {
				merged[index] = cookie
				continue
			}
			indexByKey[key] = len(merged)
			merged = append(merged, cookie)
		}
	}

	return merged
}
