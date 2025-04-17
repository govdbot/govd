package pinterest

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"govd/enums"
	"govd/models"
	"govd/util"

	"github.com/bytedance/sonic"
)

const (
	pinResourceEndpoint = "https://www.pinterest.com/resource/PinResource/get/"
	shortenerAPIFormat  = "https://api.pinterest.com/url_shortener/%s/redirect/"
)

var (
	httpSession = util.GetHTTPSession()
	validHost   = []string{
		"com", "fr", "de", "ch", "jp", "cl", "ca", "it", "co\\.uk", "nz", "ru", "com\\.au",
		"at", "pt", "co\\.kr", "es", "com\\.mx", "dk", "ph", "th", "com\\.uy", "co", "nl",
		"info", "kr", "ie", "vn", "com\\.vn", "ec", "mx", "in", "pe", "co\\.at", "hu",
		"co\\.in", "co\\.nz", "id", "com\\.ec", "com\\.py", "tw", "be", "uk", "com\\.bo", "com\\.pe",
	}
	validHostRegex     = strings.Join(validHost, "|")
	validURLPattern    = `https?://(?:[^/]+\.)?pinterest\.(` + validHostRegex + `)/pin/(?:[\w-]+--)?(?P<id>\d+)`
	pinValidURLPattern = `https?://(www\.)?pin\.(` + validHostRegex + `)/(?P<id>\w+)`
)

var ShortExtractor = &models.Extractor{
	Name:       "Pinterest (Short)",
	CodeName:   "pinterest:short",
	Type:       enums.ExtractorTypeSingle,
	Category:   enums.ExtractorCategorySocial,
	URLPattern: regexp.MustCompile(pinValidURLPattern),
	Host: func() []string {
		var domains []string
		for _, domain := range validHost {
			domains = append(domains, "pin."+domain)
		}
		return domains
	}(),
	IsRedirect: true,

	Run: func(ctx *models.DownloadContext) (*models.ExtractorResponse, error) {
		shortURL := fmt.Sprintf(shortenerAPIFormat, ctx.MatchedContentID)
		location, err := util.GetLocationURL(shortURL, "")
		if err != nil {
			return nil, fmt.Errorf("failed to get real url: %w", err)
		}
		return &models.ExtractorResponse{
			URL: location,
		}, nil
	},
}

var Extractor = &models.Extractor{
	Name:       "Pinterest",
	CodeName:   "pinterest",
	Type:       enums.ExtractorTypeSingle,
	Category:   enums.ExtractorCategorySocial,
	URLPattern: regexp.MustCompile(validURLPattern),
	Host: func() []string {
		var domains []string
		for _, domain := range validHost {
			domains = append(domains, "pinterest."+domain)
			domains = append(domains, domain+".pinterest.com")
		}
		return domains
	}(),

	Run: func(ctx *models.DownloadContext) (*models.ExtractorResponse, error) {
		media, err := ExtractPinMedia(ctx)
		if err != nil {
			return nil, err
		}
		return &models.ExtractorResponse{
			MediaList: media,
		}, nil
	},
}

func ExtractPinMedia(ctx *models.DownloadContext) ([]*models.Media, error) {
	pinID := ctx.MatchedContentID
	contentURL := ctx.MatchedContentURL

	pinData, err := GetPinData(pinID)
	if err != nil {
		return nil, err
	}

	media := ctx.Extractor.NewMedia(pinID, contentURL)
	media.SetCaption(pinData.Title)

	if pinData.Videos != nil && pinData.Videos.VideoList != nil {
		formats, err := ParseVideoObject(pinData.Videos)
		if err != nil {
			return nil, err
		}
		for _, format := range formats {
			media.AddFormat(format)
		}
		return []*models.Media{media}, nil
	}

	if pinData.StoryPinData != nil && len(pinData.StoryPinData.Pages) > 0 {
		for _, page := range pinData.StoryPinData.Pages {
			for _, block := range page.Blocks {
				if block.BlockType == 3 && block.Video != nil { // blockType 3 = Video
					formats, err := ParseVideoObject(block.Video)
					if err != nil {
						return nil, err
					}
					for _, format := range formats {
						media.AddFormat(format)
					}
					return []*models.Media{media}, nil
				}
			}
		}
	}

	if pinData.Images != nil && pinData.Images.Orig != nil {
		imageURL := pinData.Images.Orig.URL
		media.AddFormat(&models.MediaFormat{
			FormatID: "photo",
			Type:     enums.MediaTypePhoto,
			URL:      []string{imageURL},
		})
		return []*models.Media{media}, nil
	} else if pinData.StoryPinData != nil && len(pinData.StoryPinData.Pages) > 0 {
		for _, page := range pinData.StoryPinData.Pages {
			if page.Image != nil && page.Image.Images.Originals != nil {
				media.AddFormat(&models.MediaFormat{
					FormatID: "photo",
					Type:     enums.MediaTypePhoto,
					URL:      []string{page.Image.Images.Originals.URL},
				})
				return []*models.Media{media}, nil
			}
		}
	}

	if pinData.Embed != nil && pinData.Embed.Type == "gif" {
		media.AddFormat(&models.MediaFormat{
			FormatID:   "gif",
			Type:       enums.MediaTypeVideo,
			VideoCodec: enums.MediaCodecAVC,
			URL:        []string{pinData.Embed.Src},
		})
		return []*models.Media{media}, nil
	}

	return nil, fmt.Errorf("no media found for pin ID: %s", pinID)
}

func GetPinData(pinID string) (*PinData, error) {
	params := BuildPinRequestParams(pinID)

	req, err := http.NewRequest(http.MethodGet, pinResourceEndpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	q := req.URL.Query()
	for key, value := range params {
		q.Add(key, value)
	}
	req.URL.RawQuery = q.Encode()
	req.Header.Set("User-Agent", util.ChromeUA)

	// fix 403 error
	req.Header.Set("X-Pinterest-PWS-Handler", "www/[username].js")

	resp, err := httpSession.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("bad response: %s", resp.Status)
	}

	var pinResponse PinResponse
	decoder := sonic.ConfigFastest.NewDecoder(resp.Body)
	err = decoder.Decode(&pinResponse)
	if err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &pinResponse.ResourceResponse.Data, nil
}
