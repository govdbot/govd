package twitter

import (
	"fmt"
	"govd/enums"
	"govd/models"
	"govd/util"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/bytedance/sonic"
)

const authToken = "AAAAAAAAAAAAAAAAAAAAANRILgAAAAAAnNwIzUejRCOuH5E6I8xnZz4puTs%3D1Zv7ttfk8LF81IUq16cHjhLTvJu4FA33AGWWjCpTnA"

var resolutionRegex = regexp.MustCompile(`(\d+)x(\d+)`)

func BuildAPIHeaders(cookies []*http.Cookie) map[string]string {
	var csrfToken string
	for _, cookie := range cookies {
		if cookie.Name == "ct0" {
			csrfToken = cookie.Value
			break
		}
	}
	if csrfToken == "" {
		return nil
	}
	headers := map[string]string{
		"authorization":             "Bearer " + authToken,
		"user-agent":                util.ChromeUA,
		"x-twitter-auth-type":       "OAuth2Session",
		"x-twitter-client-language": "en",
		"x-twitter-active-user":     "yes",
	}

	if csrfToken != "" {
		headers["x-csrf-token"] = csrfToken
	}

	return headers
}

func BuildAPIQuery(tweetID string) map[string]string {
	variables := map[string]interface{}{
		"focalTweetId":                           tweetID,
		"includePromotedContent":                 true,
		"with_rux_injections":                    false,
		"withBirdwatchNotes":                     true,
		"withCommunity":                          true,
		"withDownvotePerspective":                false,
		"withQuickPromoteEligibilityTweetFields": true,
		"withReactionsMetadata":                  false,
		"withReactionsPerspective":               false,
		"withSuperFollowsTweetFields":            true,
		"withSuperFollowsUserFields":             true,
		"withV2Timeline":                         true,
		"withVoice":                              true,
	}

	features := map[string]interface{}{
		"graphql_is_translatable_rweb_tweet_is_translatable_enabled":              false,
		"interactive_text_enabled":                                                true,
		"responsive_web_edit_tweet_api_enabled":                                   true,
		"responsive_web_enhance_cards_enabled":                                    true,
		"responsive_web_graphql_timeline_navigation_enabled":                      false,
		"responsive_web_text_conversations_enabled":                               false,
		"responsive_web_uc_gql_enabled":                                           true,
		"standardized_nudges_misinfo":                                             true,
		"tweet_with_visibility_results_prefer_gql_limited_actions_policy_enabled": false,
		"tweetypie_unmention_optimization_enabled":                                true,
		"unified_cards_ad_metadata_container_dynamic_card_content_query_enabled":  true,
		"verified_phone_label_enabled":                                            false,
		"vibe_api_enabled":                                                        true,
	}

	variablesJSON, _ := sonic.ConfigFastest.Marshal(variables)
	featuresJSON, _ := sonic.ConfigFastest.Marshal(features)

	return map[string]string{
		"variables": string(variablesJSON),
		"features":  string(featuresJSON),
	}
}

func CleanCaption(caption string) string {
	if caption == "" {
		return ""
	}
	regex := regexp.MustCompile(`https?://t\.co/\S+`)
	return strings.TrimSpace(regex.ReplaceAllString(caption, ""))
}

func ExtractVideoFormats(media *MediaEntity) ([]*models.MediaFormat, error) {
	var formats []*models.MediaFormat

	if media.VideoInfo == nil {
		return formats, nil
	}

	duration := int64(media.VideoInfo.DurationMillis / 1000)

	for _, variant := range media.VideoInfo.Variants {
		if variant.ContentType == "video/mp4" {
			width, height := extractResolution(variant.URL)

			formats = append(formats, &models.MediaFormat{
				Type:       enums.MediaTypeVideo,
				FormatID:   fmt.Sprintf("mp4_%d", variant.Bitrate),
				URL:        []string{variant.URL},
				VideoCodec: enums.MediaCodecAVC,
				AudioCodec: enums.MediaCodecAAC,
				Duration:   duration,
				Thumbnail:  []string{media.MediaURLHTTPS},
				Width:      width,
				Height:     height,
				Bitrate:    int64(variant.Bitrate),
			})
		}
	}

	return formats, nil
}

func extractResolution(url string) (int64, int64) {
	matches := resolutionRegex.FindStringSubmatch(url)
	if len(matches) >= 3 {
		width, _ := strconv.ParseInt(matches[1], 10, 64)
		height, _ := strconv.ParseInt(matches[2], 10, 64)
		return width, height
	}
	return 0, 0
}

func FindTweetData(resp *APIResponse, tweetID string) (*Tweet, error) {
	instructions := resp.Data.ThreadedConversationWithInjectionsV2.Instructions
	if len(instructions) == 0 {
		return nil, fmt.Errorf("tweet data missing")
	}

	entries := instructions[0].Entries
	entryID := "tweet-" + tweetID

	for _, entry := range entries {
		if entry.EntryID == entryID {
			result := entry.Content.ItemContent.TweetResults.Result

			if result.Tweet != nil {
				return result.Tweet, nil
			}

			if result.Legacy != nil {
				return result.Legacy, nil
			}

			return nil, fmt.Errorf("invalid tweet data")
		}
	}

	return nil, fmt.Errorf("tweet not found")
}
