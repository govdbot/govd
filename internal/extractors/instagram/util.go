package instagram

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/govdbot/govd/internal/database"
	"github.com/govdbot/govd/internal/logger"
	"github.com/govdbot/govd/internal/models"
	"github.com/govdbot/govd/internal/networking"
	"github.com/govdbot/govd/internal/util"

	"github.com/bytedance/sonic"
	"github.com/titanous/json5"
)

const (
	graphQLEndpoint = "https://www.instagram.com/graphql/query/"
	polarisAction   = "PolarisPostActionLoadPostQueryQuery"

	igramHostname = "api-wh.igram.world"
	igramAPIBase  = "api.igram.world"
	igramHMACKey  = "75f2d70d3724f98e4a7d1ffd0ba9cfd907f3ae2632ee159980e2c521bff62358"
	igramStaticTS = 1771418815381 // parseInt("mls10xp1", 36)
)

var (
	embedPattern = regexp.MustCompile(
		`new ServerJS\(\)\);s\.handle\(({.*})\);requireLazy`)

	webHeaders = map[string]string{
		"Accept":                    "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.7",
		"Accept-Language":           "en-GB,en;q=0.9",
		"Cache-Control":             "max-age=0",
		"DNT":                       "1",
		"Priority":                  "u=0, i",
		"Sec-Ch-Ua":                 `Chromium";v="124", "Google Chrome";v="124", "Not-A.Brand";v="99"`,
		"Sec-Ch-Ua-Mobile":          "?0",
		"Sec-Ch-Ua-Platform":        "macOS",
		"Sec-Fetch-Dest":            "document",
		"Sec-Fetch-Mode":            "navigate",
		"Sec-Fetch-Site":            "none",
		"Sec-Fetch-User":            "?1",
		"Upgrade-Insecure-Requests": "1",
		"X-IG-App-ID":             "936619743392459",
		"X-IG-WWW-Claim":          "h=1",
		"X-Requested-With":        "XMLHttpRequest",
	}

	igramHeaders = map[string]string{
		"Referer": "https://igram.world/",
	}
)

func ParseGQLMedia(ctx *models.ExtractorContext, data *Media) (*models.Media, error) {
	var caption string
	if data.EdgeMediaToCaption != nil && len(data.EdgeMediaToCaption.Edges) > 0 {
		caption = data.EdgeMediaToCaption.Edges[0].Node.Text
	} else if data.Caption != "" {
		caption = data.Caption
	}

	media := ctx.NewMedia()
	media.SetCaption(caption)

	switch data.Typename {
	case "GraphVideo", "XDTGraphVideo":
		item := media.NewItem()
		formats := &models.MediaFormat{
			FormatID:     "video",
			Type:         database.MediaTypeVideo,
			VideoCodec:   database.MediaCodecAvc,
			AudioCodec:   database.MediaCodecAac,
			URL:          []string{data.VideoURL},
			Width:        dimsWidth(data.Dimensions),
			Height:       dimsHeight(data.Dimensions),
		}
		// Only attach a thumbnail when we have a real image URL
		// (not the video URL itself, which Telegram rejects).
		if data.DisplayURL != "" && data.DisplayURL != data.VideoURL {
			formats.ThumbnailURL = []string{data.DisplayURL}
		}
		item.AddFormats(formats)
	case "GraphImage", "XDTGraphImage":
		item := media.NewItem()
		item.AddFormats(&models.MediaFormat{
			FormatID: "image",
			Type:     database.MediaTypePhoto,
			URL:      []string{data.DisplayURL},
		})
	case "GraphSidecar", "XDTGraphSidecar":
		if data.EdgeSidecarToChildren != nil && len(data.EdgeSidecarToChildren.Edges) > 0 {
			edges := data.EdgeSidecarToChildren.Edges

			for i := range edges {
				item := media.NewItem()
				node := edges[i].Node

				switch node.Typename {
				case "GraphVideo", "XDTGraphVideo":
					item.AddFormats(&models.MediaFormat{
						FormatID:     "video",
						Type:         database.MediaTypeVideo,
						VideoCodec:   database.MediaCodecAvc,
						AudioCodec:   database.MediaCodecAac,
						URL:          []string{node.VideoURL},
						ThumbnailURL: []string{node.DisplayURL},
						Width:        dimsWidth(node.Dimensions),
						Height:       dimsHeight(node.Dimensions),
						})

				case "GraphImage", "XDTGraphImage":
					item.AddFormats(&models.MediaFormat{
						FormatID: "image",
						Type:     database.MediaTypePhoto,
						URL:      []string{node.DisplayURL},
					})
				}
			}
		}
	}

	return media, nil
}

func ParseEmbedGQL(body []byte) (*Media, error) {
	match := embedPattern.FindSubmatch(body)
	if len(match) < 2 {
		return nil, fmt.Errorf("gql json not found")
	}
	jsonData := match[1]

	var data map[string]any
	if err := json5.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON: %w", err)
	}
	igCtx := util.TraverseJSON(data, "contextJSON")
	if igCtx == nil {
		return nil, fmt.Errorf("contextJSON not found")
	}
	var ctxJSON ContextJSON
	switch v := igCtx.(type) {
	case string:
		if err := json5.Unmarshal([]byte(v), &ctxJSON); err != nil {
			return nil, fmt.Errorf("failed to unmarshal contextJSON: %w", err)
		}
	default:
		return nil, fmt.Errorf("unexpected type for contextJSON: %T", v)
	}
	if ctxJSON.GqlData == nil {
		return nil, fmt.Errorf("gql_data not found")
	}
	if ctxJSON.GqlData.ShortcodeMedia == nil {
		return nil, fmt.Errorf("shortcode_media not found")
	}
	return ctxJSON.GqlData.ShortcodeMedia, nil
}

func IGramBodyFromURL(contentURL string) (io.Reader, error) {
	return igramBuildPayload(map[string]string{
		"target_url": contentURL,
	})
}

func IGramBodyFromParams(params map[string]string) (io.Reader, error) {
	return igramBuildPayload(params)
}

func igramBuildPayload(urlParams map[string]string) (io.Reader, error) {
	nowMs := time.Now().UnixMilli()
	serverMs := getIGramServerTime()

	drift := serverMs - nowMs
	var correction int64
	if drift >= 60000 || drift <= -60000 {
		correction = drift
	}
	ts := nowMs + correction

	// partial payload fields that get signed
	partial := map[string]any{
		"_sc": 0,
		"_ef": 0,
		"_df": 0,
	}
	for k, v := range urlParams {
		partial[k] = v
	}

	sig, err := igramSign(partial, ts)
	if err != nil {
		return nil, err
	}

	// assemble final payload
	final := make(map[string]any, len(partial)+5)
	for k, v := range partial {
		final[k] = v
	}
	final["ts"] = ts
	final["_ts"] = igramStaticTS
	final["_tsc"] = correction
	final["_sv"] = 2
	final["_s"] = sig

	jsonBytes, err := sonic.ConfigFastest.Marshal(final)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payload: %w", err)
	}

	return strings.NewReader(string(jsonBytes)), nil
}

func igramSign(partial map[string]any, ts int64) (string, error) {
	// sonic.ConfigStd sorts map keys alphabetically, matching
	// the signing: JSON.stringify(sorted_partial) + String(ts)
	jsonBytes, err := sonic.ConfigStd.Marshal(partial)
	if err != nil {
		return "", fmt.Errorf("failed to marshal partial payload: %w", err)
	}

	data := string(jsonBytes) + strconv.FormatInt(ts, 10)

	keyBytes, err := hex.DecodeString(igramHMACKey)
	if err != nil {
		return "", fmt.Errorf("failed to decode HMAC key: %w", err)
	}

	mac := hmac.New(sha256.New, keyBytes)
	mac.Write([]byte(data))
	return hex.EncodeToString(mac.Sum(nil)), nil
}

func getIGramServerTime() int64 {
	apiURL := fmt.Sprintf("https://%s/msec", igramAPIBase)
	resp, err := http.Get(apiURL)
	if err != nil {
		return time.Now().UnixMilli()
	}
	defer resp.Body.Close()

	var result struct {
		Msec float64 `json:"msec"`
	}
	decoder := sonic.ConfigFastest.NewDecoder(resp.Body)
	if err := decoder.Decode(&result); err != nil {
		return time.Now().UnixMilli()
	}
	return int64(result.Msec * 1000)
}

func ParseIGramResponse(body []byte) (*IGramResponse, error) {
	// try to unmarshal as a single IGramMedia and then as a slice
	var media IGramMedia

	if err := sonic.ConfigFastest.Unmarshal(body, &media); err != nil {
		// try with slice
		var mediaList []*IGramMedia
		if err := sonic.ConfigFastest.Unmarshal(body, &mediaList); err != nil {
			return nil, fmt.Errorf("failed to decode response: %w", err)
		}
		return &IGramResponse{
			Items: mediaList,
		}, nil
	}
	if media.Success != nil && !(*media.Success) {
		return nil, util.ErrUnavailable
	}
	return &IGramResponse{
		Items: []*IGramMedia{&media},
	}, nil
}

func GetCDNURL(contentURL string) (string, error) {
	parsedURL, err := url.Parse(contentURL)
	if err != nil {
		return "", fmt.Errorf("can't parse igram URL: %w", err)
	}
	queryParams, err := url.ParseQuery(parsedURL.RawQuery)
	if err != nil {
		return "", fmt.Errorf("can't unescape igram URL: %w", err)
	}
	cdnURL := queryParams.Get("uri")
	return cdnURL, nil
}

func GetGQLData(ctx *models.ExtractorContext) (*GraphQLData, error) {
	graphHeaders, body, err := BuildGQLData()
	if err != nil {
		return nil, fmt.Errorf("failed to build GQL data: %w", err)
	}
	formData := url.Values{}
	for key, value := range body {
		formData.Set(key, value)
	}
	formData.Set("fb_api_caller_class", "RelayModern")
	formData.Set("fb_api_req_friendly_name", polarisAction)
	variables := map[string]any{
		"shortcode":               ctx.ContentID,
		"fetch_tagged_user_count": nil,
		"hoisted_comment_id":      nil,
		"hoisted_reply_id":        nil,
	}
	variablesJSON, err := sonic.ConfigFastest.Marshal(variables)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal variables: %w", err)
	}
	formData.Set("variables", string(variablesJSON))
	formData.Set("server_timestamps", "true")
	formData.Set("doc_id", "8845758582119845") // idk what this is

	for key, value := range webHeaders {
		graphHeaders[key] = value
	}
	resp, err := ctx.Fetch(
		http.MethodPost,
		graphQLEndpoint,
		&networking.RequestParams{
			Headers: graphHeaders,
			Body:    strings.NewReader(formData.Encode()),
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	logger.WriteFile("iggql_api_response", resp)

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("invalid response code: %s", resp.Status)
	}
	var response GraphQLResponse
	decoder := sonic.ConfigFastest.NewDecoder(resp.Body)
	if err := decoder.Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}
	if response.Data == nil {
		return nil, fmt.Errorf("data is nil")
	}
	if response.Status != "ok" {
		return nil, fmt.Errorf("status is not ok: %s", response.Status)
	}
	if response.Data.ShortcodeMedia == nil {
		return nil, fmt.Errorf("shortcode_media is nil")
	}
	return response.Data, nil
}

func BuildGQLData() (map[string]string, map[string]string, error) {
	const (
		domain                = "www"
		requestID             = "b"
		clientCapabilityGrade = "EXCELLENT"
		sessionInternalID     = "7436540909012459023"
		apiVersion            = "1"
		rolloutHash           = "1019933358"
		appID                 = "936619743392459"
		bloksVersionID        = "6309c8d03d8a3f47a1658ba38b304a3f837142ef5f637ebf1f8f52d4b802951e"
		asbdID                = "129477"
		hiddenState           = "20126.HYP:instagram_web_pkg.2.1...0"
		loggedIn              = "0"
		cometRequestID        = "7"
		appVersion            = "0"
		pixelRatio            = "2"
		buildType             = "trunk"
	)
	session := "::" + util.RandomAlphaString(6)
	sessionData := util.RandomBase64(8)
	csrfToken := util.RandomBase64(32)
	deviceID := util.RandomBase64(24)
	machineID := util.RandomBase64(24)
	dynamicFlags := util.RandomBase64(154)
	clientSessionRnd := util.RandomBase64(154)
	jazoestBig, err := rand.Int(rand.Reader, big.NewInt(10000))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate jazoest: %w", err)
	}
	jazoest := strconv.FormatInt(jazoestBig.Int64()+1, 10)
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)
	cookies := []string{
		"csrftoken=" + csrfToken,
		"ig_did=" + deviceID,
		"wd=1280x720",
		"dpr=2",
		"mid=" + machineID,
		"ig_nrcb=1",
	}
	headers := map[string]string{
		"x-ig-app-id":        appID,
		"X-FB-LSD":           sessionData,
		"X-CSRFToken":        csrfToken,
		"X-Bloks-Version-Id": bloksVersionID,
		"x-asbd-id":          asbdID,
		"cookie":             strings.Join(cookies, "; "),
		"Content-Type":       "application/x-www-form-urlencoded",
		"X-FB-Friendly-Name": polarisAction,
	}
	body := map[string]string{
		"__d":         domain,
		"__a":         apiVersion,
		"__s":         session,
		"__hs":        hiddenState,
		"__req":       requestID,
		"__ccg":       clientCapabilityGrade,
		"__rev":       rolloutHash,
		"__hsi":       sessionInternalID,
		"__dyn":       dynamicFlags,
		"__csr":       clientSessionRnd,
		"__user":      loggedIn,
		"__comet_req": cometRequestID,
		"libav":       appVersion,
		"dpr":         pixelRatio,
		"lsd":         sessionData,
		"jazoest":     jazoest,
		"__spin_r":    rolloutHash,
		"__spin_b":    buildType,
		"__spin_t":    timestamp,
	}
	return headers, body, nil
}

func GetBestCandidate(candidates []*Candidates) *Candidates {
	if len(candidates) == 0 {
		return nil
	}
	best := candidates[0]
	for _, candidate := range candidates {
		if candidate.Width > best.Width {
			best = candidate
		}
	}
	return best
}

func GetBestVideoVersion(versions []*VideoVersions) *VideoVersions {
	if len(versions) == 0 {
		return nil
	}
	best := versions[0]
	for _, version := range versions {
		if version.Width > best.Width {
			best = version
		}
	}
	return best
}

// GetWebpageMedia fetches Instagram media via the public GraphQL
// `web_info` endpoint (instaloader PR #2706 approach). This returns the full
// media node — including carousel/album children — as JSON, so no browser /
// JS execution is required. Works with a normal authenticated cookie.
func GetWebpageMedia(ctx *models.ExtractorContext) (*models.Media, error) {
	// Load Instagram cookies (private/cookies/instagram.txt) so authenticated
	// posts (albums/carousels) return media data instead of a block page.
	cookies := util.GetExtractorCookies(ctx.Extractor.ID)

	// Build the GraphQL query URL. We use the doc_id that returns
	// `xdt_api__v1__media__shortcode__web_info` (instaloader PR #2706).
	variables := fmt.Sprintf(
		`{"shortcode":%q,"__relay_internal__pv__PolarisAIGMMediaWebLabelEnabledrelayprovider":false}`,
		ctx.ContentID,
	)
	apiURL := fmt.Sprintf(
		"https://www.instagram.com/graphql/query/?doc_id=27128499623469141&variables=%s",
		url.QueryEscape(variables),
	)

	// Minimal headers — matches what works via curl. The full webHeaders
	// (Sec-Fetch-*, Upgrade-Insecure-Requests) makes IG reject the GraphQL
	// request from the bot's client, so we keep it lean here.
	gqlHeaders := map[string]string{
		"Accept":          "*/*",
		"X-IG-App-ID":     "936619743392459",
		"X-Requested-With": "XMLHttpRequest",
		"Referer":         "https://www.instagram.com/",
	}

	resp, err := ctx.Fetch(
		http.MethodGet,
		apiURL,
		&networking.RequestParams{
			Headers: gqlHeaders,
			Cookies: cookies,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get media via graphql: %s", resp.Status)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Parse the JSON response.
	var root struct {
		Data struct {
			WebInfo struct {
				Items []json.RawMessage `json:"items"`
			} `json:"xdt_api__v1__media__shortcode__web_info"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &root); err != nil {
		return nil, fmt.Errorf("failed to parse graphql response: %w", err)
	}
	items := root.Data.WebInfo.Items
	if len(items) == 0 {
		return nil, fmt.Errorf("no media items returned (blocked or not found)")
	}

	// Use the first item (posts are single items).
	item := items[0]

	var node struct {
		Typename        string `json:"__typename"`
		ID              string `json:"id"`
		Shortcode       string `json:"code"`
		IsVideo         bool   `json:"is_video"`
		OriginalWidth   int    `json:"original_width"`
		OriginalHeight  int    `json:"original_height"`
		Caption         *struct {
			Text string `json:"text"`
		} `json:"caption"`
		VideoVersions   []struct {
			URL    string `json:"url"`
			Type   int    `json:"type"`
			Width  int    `json:"width"`
			Height int    `json:"height"`
		} `json:"video_versions"`
		ImageVersions2 struct {
			Candidates []struct {
				URL    string `json:"url"`
				Width  int    `json:"width"`
				Height int    `json:"height"`
			} `json:"candidates"`
		} `json:"image_versions2"`
		CarouselMedia []struct {
			IsVideo       bool `json:"is_video"`
			OriginalWidth  int `json:"original_width"`
			OriginalHeight int `json:"original_height"`
			VideoVersions []struct {
				URL string `json:"url"`
			} `json:"video_versions"`
			ImageVersions2 struct {
				Candidates []struct {
					URL string `json:"url"`
				} `json:"candidates"`
			} `json:"image_versions2"`
		} `json:"carousel_media"`
	}
	if err := json.Unmarshal(item, &node); err != nil {
		return nil, fmt.Errorf("failed to parse media item: %w", err)
	}

	// DEBUG (temporary): log what the web_info endpoint actually returned
	// for this shortcode so we can see why a reel is sent as a photo.
	logger.L.Infow("IG WEBPAGE DEBUG",
		"shortcode", ctx.ContentID,
		"typename", node.Typename,
		"is_video", node.IsVideo,
		"video_versions_len", len(node.VideoVersions),
		"image_candidates_len", len(node.ImageVersions2.Candidates),
		"carousel_len", len(node.CarouselMedia),
	)

	var caption string
	if node.Caption != nil {
		caption = node.Caption.Text
	}

	// Carousel / album: build child medias using EdgeSidecarToChildren
	// (the structure ParseGQLMedia already understands).
	if len(node.CarouselMedia) > 0 {
		edges := make([]*EdgeNode, 0, len(node.CarouselMedia))
		for _, child := range node.CarouselMedia {
			childMedia := &Media{}
			// NOTE: the web_info endpoint can return is_video:null even for
			// real videos, so rely on the presence of video_versions instead.
			if len(child.VideoVersions) > 0 {
				childMedia.Typename = "XDTGraphVideo"
				childMedia.IsVideo = true
				childMedia.VideoURL = child.VideoVersions[0].URL
				// web_info exposes original_width/height, not a dimensions object
				if child.OriginalWidth > 0 || child.OriginalHeight > 0 {
					childMedia.Dimensions = &Dimensions{Width: int32(child.OriginalWidth), Height: int32(child.OriginalHeight)}
				}
			} else if len(child.ImageVersions2.Candidates) > 0 {
				childMedia.Typename = "XDTGraphImage"
				childMedia.DisplayURL = child.ImageVersions2.Candidates[0].URL
			} else {
				continue
			}
			edges = append(edges, &EdgeNode{Node: childMedia})
		}
		if len(edges) == 0 {
			return nil, fmt.Errorf("carousel returned no usable media")
		}
		album := &Media{
			Typename:              "XDTGraphSidecar",
			Caption:               caption,
			ID:                    ctx.ContentID,
			Shortcode:             ctx.ContentID,
			EdgeSidecarToChildren: &EdgeSidecarToChildren{Edges: edges},
		}
		return ParseGQLMedia(ctx, album)
	}

	// Single video.
	// NOTE: the web_info endpoint can return is_video:null even for real
	// videos (e.g. some reels), so detect video by the presence of
	// video_versions rather than the is_video boolean.
	if len(node.VideoVersions) > 0 {
		// Attach the still frame as a thumbnail preview (Telegram shows it
		// before playback). image_versions2[0] is the poster frame.
		thumb := ""
		if len(node.ImageVersions2.Candidates) > 0 {
			thumb = node.ImageVersions2.Candidates[0].URL
		}
		// The web_info endpoint does not include a `dimensions` object for
		// reels/videos; it exposes original_width/original_height instead.
		// Build Dimensions so ParseGQLMedia does not dereference a nil pointer.
		var dims *Dimensions
		if node.OriginalWidth > 0 || node.OriginalHeight > 0 {
			dims = &Dimensions{Width: int32(node.OriginalWidth), Height: int32(node.OriginalHeight)}
		}
		media := &Media{
			Typename:   "XDTGraphVideo",
			IsVideo:    true,
			VideoURL:   node.VideoVersions[0].URL,
			DisplayURL: thumb,
			Caption:    caption,
			ID:         ctx.ContentID,
			Shortcode:  ctx.ContentID,
			Dimensions: dims,
		}
		return ParseGQLMedia(ctx, media)
	}

	// Single photo.
	if len(node.ImageVersions2.Candidates) > 0 {
		praw := node.ImageVersions2.Candidates[0].URL
		media := &Media{
			Typename:   "XDTGraphImage",
			IsVideo:    false,
			DisplayURL: praw,
			Caption:    caption,
			ID:         ctx.ContentID,
			Shortcode:  ctx.ContentID,
		}
		return ParseGQLMedia(ctx, media)
	}

	return nil, fmt.Errorf("no formats found for media item")
}

// dimsWidth / dimsHeight safely read a possibly-nil *Dimensions so callers
// don't panic on reels/videos whose web_info response omits the dimensions
// object (it exposes original_width/original_height instead).
func dimsWidth(d *Dimensions) int32 {
	if d == nil {
		return 0
	}
	return d.Width
}

func dimsHeight(d *Dimensions) int32 {
	if d == nil {
		return 0
	}
	return d.Height
}
