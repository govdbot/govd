package instagram

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"govd/enums"
	"govd/models"
	"govd/util"

	"github.com/bytedance/sonic"
	"github.com/pkg/errors"
	"github.com/titanous/json5"
)

const (
	graphQLEndpoint = "https://www.instagram.com/graphql/query/"
	polarisAction   = "PolarisPostActionLoadPostQueryQuery"

	igramHostname  = "api.igram.world"
	igramKey       = "aaeaf2805cea6abef3f9d2b6a666fce62fd9d612a43ab772bb50ce81455112e0"
	igramTimestamp = "1742201548873"
)

var (
	embedPattern = regexp.MustCompile(
		`new ServerJS\(\)\);s\.handle\(({.*})\);requireLazy`)

	igHeaders = map[string]string{
		"Accept":                    "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.7",
		"Accept-Language":           "en-GB,en;q=0.9",
		"Cache-Control":             "max-age=0",
		"Dnt":                       "1",
		"Priority":                  "u=0, i",
		"Sec-Ch-Ua":                 `Chromium";v="124", "Google Chrome";v="124", "Not-A.Brand";v="99`,
		"Sec-Ch-Ua-Mobile":          "?0",
		"Sec-Ch-Ua-Platform":        "macOS",
		"Sec-Fetch-Dest":            "document",
		"Sec-Fetch-Mode":            "navigate",
		"Sec-Fetch-Site":            "none",
		"Sec-Fetch-User":            "?1",
		"Upgrade-Insecure-Requests": "1",
		"User-Agent":                util.ChromeUA,
	}
)

func ParseGQLMedia(
	ctx *models.DownloadContext,
	data *Media,
) ([]*models.Media, error) {
	var caption string
	if data.EdgeMediaToCaption != nil && len(data.EdgeMediaToCaption.Edges) > 0 {
		caption = data.EdgeMediaToCaption.Edges[0].Node.Text
	}

	contentID := ctx.MatchedContentID
	contentURL := ctx.MatchedContentURL

	switch data.Typename {
	case "GraphVideo", "XDTGraphVideo":
		media := ctx.Extractor.NewMedia(contentID, contentURL)
		media.SetCaption(caption)

		media.AddFormat(&models.MediaFormat{
			FormatID:   "video",
			Type:       enums.MediaTypeVideo,
			VideoCodec: enums.MediaCodecAVC,
			AudioCodec: enums.MediaCodecAAC,
			URL:        []string{data.VideoURL},
			Thumbnail:  []string{data.DisplayURL},
			Width:      int64(data.Dimensions.Width),
			Height:     int64(data.Dimensions.Height),
		})

		return []*models.Media{media}, nil
	case "GraphImage", "XDTGraphImage":
		media := ctx.Extractor.NewMedia(contentID, contentURL)
		media.SetCaption(caption)

		media.AddFormat(&models.MediaFormat{
			FormatID: "image",
			Type:     enums.MediaTypePhoto,
			URL:      []string{data.DisplayURL},
		})

		return []*models.Media{media}, nil
	case "GraphSidecar", "XDTGraphSidecar":
		if data.EdgeSidecarToChildren != nil && len(data.EdgeSidecarToChildren.Edges) > 0 {
			edges := data.EdgeSidecarToChildren.Edges
			mediaList := make([]*models.Media, 0, len(edges))

			for i := range edges {
				node := edges[i].Node
				media := ctx.Extractor.NewMedia(contentID, contentURL)
				media.SetCaption(caption)

				switch node.Typename {
				case "GraphVideo", "XDTGraphVideo":
					media.AddFormat(&models.MediaFormat{
						FormatID:   "video",
						Type:       enums.MediaTypeVideo,
						VideoCodec: enums.MediaCodecAVC,
						AudioCodec: enums.MediaCodecAAC,
						URL:        []string{node.VideoURL},
						Thumbnail:  []string{node.DisplayURL},
						Width:      int64(node.Dimensions.Width),
						Height:     int64(node.Dimensions.Height),
					})

				case "GraphImage", "XDTGraphImage":
					media.AddFormat(&models.MediaFormat{
						FormatID: "image",
						Type:     enums.MediaTypePhoto,
						URL:      []string{node.DisplayURL},
					})
				}

				mediaList = append(mediaList, media)
			}
			return mediaList, nil
		}
	}

	return nil, fmt.Errorf("unknown media type: %s", data.Typename)
}

func ParseEmbedGQL(
	body []byte,
) (*Media, error) {
	match := embedPattern.FindSubmatch(body)
	if len(match) < 2 {
		return nil, errors.New("failed to find JSON in response")
	}
	jsonData := match[1]

	var data map[string]any
	if err := json5.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON: %w", err)
	}
	igCtx := util.TraverseJSON(data, "contextJSON")
	if igCtx == nil {
		return nil, errors.New("contextJSON not found in data")
	}
	var ctxJSON ContextJSON
	switch v := igCtx.(type) {
	case string:
		if err := json5.Unmarshal([]byte(v), &ctxJSON); err != nil {
			return nil, fmt.Errorf("failed to unmarshal contextJSON: %w", err)
		}
	default:
		return nil, errors.New("contextJSON is not a string")
	}
	if ctxJSON.GqlData == nil {
		return nil, errors.New("gql_data is nil")
	}
	if ctxJSON.GqlData.ShortcodeMedia == nil {
		return nil, errors.New("media is nil")
	}
	return ctxJSON.GqlData.ShortcodeMedia, nil
}

func BuildIGramPayload(contentURL string) (io.Reader, error) {
	timestamp := strconv.FormatInt(time.Now().UnixMilli(), 10)
	hash := sha256.New()
	_, err := io.WriteString(
		hash,
		contentURL+timestamp+igramKey,
	)
	if err != nil {
		return nil, fmt.Errorf("error writing to SHA256 hash: %w", err)
	}
	secretBytes := hash.Sum(nil)
	secretString := hex.EncodeToString(secretBytes)
	secretString = strings.ToLower(secretString)
	payload := map[string]string{
		"url":  contentURL,
		"ts":   timestamp,
		"_ts":  igramTimestamp,
		"_tsc": "0", // ?
		"_s":   secretString,
	}
	parsedPayload, err := sonic.ConfigFastest.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("error marshalling payload: %w", err)
	}
	reader := bytes.NewReader(parsedPayload)
	return reader, nil
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

func GetGQLData(
	ctx *models.DownloadContext,
	shortcode string,
) (*GraphQLData, error) {
	session := util.GetHTTPClient(ctx.Extractor.CodeName)
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
		"shortcode":               shortcode,
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
	req, err := http.NewRequest(
		http.MethodPost,
		graphQLEndpoint,
		strings.NewReader(formData.Encode()),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	for key, value := range igHeaders {
		req.Header.Set(key, value)
	}
	for key, value := range graphHeaders {
		req.Header.Set(key, value)
	}
	resp, err := session.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("invalid response code: %s", resp.Status)
	}
	var response GraphQLResponse
	decoder := sonic.ConfigFastest.NewDecoder(resp.Body)
	if err := decoder.Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}
	if response.Data == nil {
		return nil, errors.New("data is nil")
	}
	if response.Status != "ok" {
		return nil, fmt.Errorf("status is not ok: %s", response.Status)
	}
	if response.Data.ShortcodeMedia == nil {
		return nil, errors.New("media is nil")
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
		"av":          appVersion,
		"dpr":         pixelRatio,
		"lsd":         sessionData,
		"jazoest":     jazoest,
		"__spin_r":    rolloutHash,
		"__spin_b":    buildType,
		"__spin_t":    timestamp,
	}
	return headers, body, nil
}
