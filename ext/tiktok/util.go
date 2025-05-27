package tiktok

import (
	"crypto/rand"
	"fmt"
	"io"
	"maps"
	"math/big"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/bytedance/sonic"

	"github.com/govdbot/govd/enums"
	"github.com/govdbot/govd/ext/tiktok/signer"
	"github.com/govdbot/govd/logger"
	"github.com/govdbot/govd/models"
	"github.com/govdbot/govd/util"
	"github.com/govdbot/govd/util/networking"

	"github.com/google/uuid"
)

var (
	universalDataPattern = regexp.MustCompile(`<script[^>]+\bid="__UNIVERSAL_DATA_FOR_REHYDRATION__"[^>]*>(.*?)<\/script>`)

	appHeaders = map[string]string{
		"User-Agent":   appUserAgent,
		"Accept":       "application/json",
		"Content-Type": "application/x-www-form-urlencoded",
	}
	webHeaders = map[string]string{
		"Host":            "www.tiktok.com",
		"Connection":      "keep-alive",
		"User-Agent":      "Mozilla/5.0",
		"Accept":          "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8",
		"Accept-Language": "en-us,en;q=0.5",
		"Sec-Fetch-Mode":  "navigate",
	}
)

func GetVideoWeb(ctx *models.DownloadContext) (*WebItemStruct, []*http.Cookie, error) {
	client := networking.GetExtractorHTTPClient(ctx.Extractor)
	cookies := util.GetExtractorCookies(ctx.Extractor)
	awemeID := ctx.MatchedContentID
	url := fmt.Sprintf(webBase, awemeID)
	resp, err := util.FetchPage(
		client,
		http.MethodGet,
		url,
		nil,
		webHeaders,
		cookies,
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

func GetVideoAPI(ctx *models.DownloadContext) (*AwemeDetail, error) {
	client := networking.GetExtractorHTTPClient(ctx.Extractor)
	cookies := util.GetExtractorCookies(ctx.Extractor)

	awemeID := ctx.MatchedContentID
	apiURL := fmt.Sprintf(
		"https://%s/aweme/v1/aweme/detail/",
		apiHostname,
	)
	queryParams, err := BuildAPIQuery()
	if err != nil {
		return nil, fmt.Errorf("failed to build api query: %w", err)
	}
	postData := BuildPostData(awemeID)
	postDataReader := strings.NewReader(postData)

	// generate signed headers
	headers, err := signer.Sign(
		queryParams,
		postData,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to sign request: %w", err)
	}

	reqURL := apiURL + "?" + queryParams.Encode()
	maps.Copy(headers, appHeaders)

	resp, err := util.FetchPage(
		client,
		http.MethodPost,
		reqURL,
		postDataReader,
		headers,
		cookies,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// debugging
	logger.WriteFile("tt_api_response", resp)

	var data *Response
	decoder := sonic.ConfigFastest.NewDecoder(resp.Body)
	err = decoder.Decode(&data)
	if err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	if data.StatusCode == 2053 {
		return nil, util.ErrUnavailable
	}
	if data.AwemeDetail == nil {
		return nil, ErrAwemeDetailNil
	}
	return data.AwemeDetail, nil
}

func BuildAPIQuery() (url.Values, error) {
	requestTicket := strconv.Itoa(int(time.Now().Unix()) * 1000)
	clientDeviceID := uuid.New().String()
	versionCode, err := GetAppVersionCode(appVersion)
	if err != nil {
		return nil, fmt.Errorf("failed to get app version code: %w", err)
	}
	return url.Values{
		"device_platform":       []string{"android"},
		"os":                    []string{"android"},
		"ssmix":                 []string{"0"}, // what is this?
		"_rticket":              []string{requestTicket},
		"cdid":                  []string{clientDeviceID},
		"channel":               []string{"googleplay"},
		"aid":                   []string{appID},
		"app_name":              []string{appName},
		"version_code":          []string{versionCode},
		"version_name":          []string{appVersion},
		"manifest_version_code": []string{manifestAppVersion},
		"update_version_code":   []string{manifestAppVersion},
		"ab_version":            []string{appVersion},
		"resolution":            []string{"1080*2400"},
		"dpi":                   []string{"420"},
		"device_type":           []string{"Pixel 7"},
		"device_brand":          []string{"Google"},
		"language":              []string{"en"},
		"os_api":                []string{"29"},
		"os_version":            []string{"13"},
		"ac":                    []string{"wifi"},
		"is_pad":                []string{"0"},
		"current_region":        []string{"US"},
		"app_type":              []string{"normal"},
		"app_version":           []string{appVersion},
		"last_install_time":     []string{GetRandomInstallTime()},
		"timezone_name":         []string{"America/New_York"},
		"residence":             []string{"US"},
		"app_language":          []string{"en"},
		"timezone_offset":       []string{"-14400"},
		"host_abi":              []string{"armeabi-v7a"},
		"locale":                []string{"en"},
		"ac2":                   []string{"wifi5g"},
		"uoo":                   []string{"1"}, // what is this?
		"carrier_region":        []string{"US"},
		"build_number":          []string{appVersion},
		"region":                []string{"US"},
		"ts":                    []string{strconv.Itoa(int(time.Now().Unix()))},
		"iid":                   []string{"123"}, // installation id, unchecked
		"device_id":             []string{GetRandomDeviceID()},
		"openudid":              []string{GetRandomUdid()},
	}, nil
}

func ParsePlayAddr(
	video *Video,
	playAddr *PlayAddr,
) (*models.MediaFormat, error) {
	formatID := playAddr.URLKey
	if formatID == "" {
		return nil, ErrURLKeyNotFound
	}
	videoCodec := enums.MediaCodecHEVC
	if strings.Contains(formatID, "h264") {
		videoCodec = enums.MediaCodecAVC
	}
	videoURL := playAddr.URLList
	videoDuration := video.Duration / 1000
	videoWidth := playAddr.Width
	videoHeight := playAddr.Height
	videoCover := &video.Cover
	videoThumbnailURLs := videoCover.URLList

	return &models.MediaFormat{
		Type:       enums.MediaTypeVideo,
		FormatID:   formatID,
		URL:        videoURL,
		VideoCodec: videoCodec,
		AudioCodec: enums.MediaCodecAAC,
		Duration:   videoDuration,
		Thumbnail:  videoThumbnailURLs,
		Width:      videoWidth,
		Height:     videoHeight,
	}, nil
}

func GetRandomInstallTime() string {
	currentTime := int(time.Now().Unix())
	minOffset := big.NewInt(86400)
	maxOffset := big.NewInt(1123200)
	diff := new(big.Int).Sub(maxOffset, minOffset)
	randomOffset, _ := rand.Int(rand.Reader, diff)
	randomOffset.Add(randomOffset, minOffset)
	result := currentTime - int(randomOffset.Int64())
	return strconv.Itoa(result)
}

func GetRandomUdid() string {
	const charset = "0123456789abcdef"
	result := make([]byte, 16)

	for i := range result {
		index, _ := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		result[i] = charset[index.Int64()]
	}
	return string(result)
}

func GetRandomDeviceID() string {
	minNum := big.NewInt(7250000000000000000)
	maxNum := big.NewInt(7351147085025500000)
	diff := new(big.Int).Sub(maxNum, minNum)
	randNum, _ := rand.Int(rand.Reader, diff)
	result := new(big.Int).Add(randNum, minNum)
	return result.String()
}

func BuildPostData(awemeID string) string {
	data := url.Values{
		"aweme_id":       []string{awemeID},
		"request_source": []string{"0"},
	}
	return data.Encode()

}

func GetAppVersionCode(version string) (string, error) {
	parts := strings.Split(version, ".")

	var result strings.Builder
	for _, part := range parts {
		num, err := strconv.Atoi(part)
		if err != nil {
			return "", fmt.Errorf("failed to parse version part: %w", err)
		}
		_, err = fmt.Fprintf(&result, "%02d", num)
		if err != nil {
			return "", fmt.Errorf("failed to format version part: %w", err)
		}
	}
	return result.String(), nil
}

func ParseUniversalData(body []byte) (*WebItemStruct, error) {
	matches := universalDataPattern.FindSubmatch(body)
	if len(matches) < 2 {
		return nil, ErrUniversalDataNotFound
	}
	var data any
	err := sonic.ConfigFastest.Unmarshal(matches[1], &data)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal universal data: %w", err)
	}
	defaultScope := util.TraverseJSON(data, "__DEFAULT_SCOPE__")
	if defaultScope == nil {
		return nil, ErrUniversalDataNotFound
	}
	itemStruct := util.TraverseJSON(defaultScope, "itemStruct")
	if itemStruct == nil {
		return nil, ErrItemStructNotFound
	}

	// debugging
	logger.WriteFile("tt_item_struct", itemStruct)

	itemStructBytes, err := sonic.ConfigFastest.Marshal(itemStruct)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal item struct: %w", err)
	}

	var webItem WebItemStruct
	err = sonic.ConfigFastest.Unmarshal(itemStructBytes, &webItem)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal item struct: %w", err)
	}
	return &webItem, nil
}
