package instagram

import (
	"fmt"
	"io"
	"maps"
	"net/http"
	"regexp"

	"github.com/bytedance/sonic"
	"github.com/govdbot/govd/internal/database"
	"github.com/govdbot/govd/internal/logger"
	"github.com/govdbot/govd/internal/models"
	"github.com/govdbot/govd/internal/networking"
	"github.com/govdbot/govd/internal/util"
)

var instagramHost = []string{"instagram", "ddinstagram"}

var Extractor = &models.Extractor{
	ID:          "instagram",
	DisplayName: "Instagram",

	URLPattern: regexp.MustCompile(`https:\/\/(www\.)?(?:dd)?instagram\.com\/(reels?|p|tv)\/(?P<id>[a-zA-Z0-9_-]+)`),
	Host:       instagramHost,
	Redirect:   false,

	GetFunc: func(ctx *models.ExtractorContext) (*models.ExtractorResponse, error) {
		// Primary (and only) method: GraphQL web_info endpoint (instaloader
		// PR #2706). This returns carousel/album children as JSON with no
		// browser required. Fallbacks removed — the other methods rely on
		// revoked doc_ids / 3rd-party services that no longer work.
		media, err := GetWebpageMedia(ctx)
		if err != nil {
			return nil, err
		}
		return &models.ExtractorResponse{
			Media: media,
		}, nil
	},
}

var StoriesExtractor = &models.Extractor{
	ID:          "instagram",
	DisplayName: "Instagram Stories",

	URLPattern: regexp.MustCompile(`https:\/\/(www\.)?(?:dd)?instagram\.com\/stories\/[a-zA-Z0-9._]+\/(?P<id>\d+)`),
	Host:       instagramHost,
	Hidden:     true,

	GetFunc: func(ctx *models.ExtractorContext) (*models.ExtractorResponse, error) {
		media, err := GetIGramStory(ctx)
		return &models.ExtractorResponse{
			Media: media,
		}, err
	},
}

var ShareURLExtractor = &models.Extractor{
	ID:          "instagram",
	DisplayName: "Instagram (Share)",

	URLPattern: regexp.MustCompile(`https?:\/\/(www\.)?(?:dd)?instagram\.com\/share\/((reels?|video|s|p)\/)?(?P<id>[^\/\?]+)`),
	Host:       instagramHost,

	Redirect: true,

	GetFunc: func(ctx *models.ExtractorContext) (*models.ExtractorResponse, error) {
		redirectURL, err := ctx.FetchLocation(ctx.ContentURL, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to get url location: %w", err)
		}
		return &models.ExtractorResponse{URL: redirectURL}, nil
	},
}

func GetIGramStory(ctx *models.ExtractorContext) (*models.Media, error) {
	details, err := GetStoryFromIGram(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get story: %w", err)
	}

	if len(details.Result) == 0 {
		return nil, util.ErrUnavailable
	}
	result := details.Result[0]
	isVideo := len(result.VideoVersions) > 0

	media := ctx.NewMedia()
	item := media.NewItem()
	if isVideo {
		video := GetBestVideoVersion(result.VideoVersions)
		item.AddFormats(&models.MediaFormat{
			FormatID:   "video",
			Type:       database.MediaTypeVideo,
			URL:        []string{video.URL},
			VideoCodec: database.MediaCodecAvc,
			AudioCodec: database.MediaCodecAac,
		})
	} else {
		image := GetBestCandidate(result.ImageVersions.Candidates)
		item.AddFormats(&models.MediaFormat{
			Type:     database.MediaTypePhoto,
			FormatID: "photo",
			URL:      []string{image.URL},
		})
	}

	if len(media.Items) == 0 {
		return nil, fmt.Errorf("no media found")
	}

	return media, nil
}

func GetPostFromIGram(ctx *models.ExtractorContext) (*IGramResponse, error) {
	contentURL := "https://www.instagram.com/p/" + ctx.ContentID + "/"
	apiURL := fmt.Sprintf("https://%s/api/convert", igramHostname)
	payload, err := IGramBodyFromURL(contentURL)
	if err != nil {
		return nil, fmt.Errorf("failed to build signed payload: %w", err)
	}

	headers := map[string]string{
		"Content-Type": "application/json",
	}
	maps.Copy(headers, igramHeaders)

	resp, err := ctx.Fetch(
		http.MethodPost,
		apiURL,
		&networking.RequestParams{
			Body:    payload,
			Headers: headers,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	logger.WriteFile("ig_3party_response", resp)

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get response: %s", resp.Status)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}
	response, err := ParseIGramResponse(body)
	if err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}
	return response, nil
}

func GetStoryFromIGram(ctx *models.ExtractorContext) (*IGramStoryResponse, error) {
	apiURL := fmt.Sprintf("https://%s/api/v1/instagram/story", igramHostname)
	payload, err := IGramBodyFromParams(map[string]string{
		"url": ctx.ContentURL,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to build signed payload: %w", err)
	}

	headers := map[string]string{
		"Content-Type": "application/json",
	}
	maps.Copy(headers, igramHeaders)

	resp, err := ctx.Fetch(
		http.MethodPost,
		apiURL,
		&networking.RequestParams{
			Body:    payload,
			Headers: headers,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	logger.WriteFile("ig_story_3party_response", resp)

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get response: %s", resp.Status)
	}

	var story IGramStoryResponse
	decoder := sonic.ConfigFastest.NewDecoder(resp.Body)
	err = decoder.Decode(&story)
	if err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &story, nil
}
