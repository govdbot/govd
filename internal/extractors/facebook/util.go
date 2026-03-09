package facebook

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/govdbot/govd/internal/logger"
	"github.com/govdbot/govd/internal/models"
	"github.com/govdbot/govd/internal/networking"
)

var webHeaders = map[string]string{
	"User-Agent":                "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36",
	"Accept":                    "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8",
	"Accept-Language":           "en-US,en;q=0.5",
	"Sec-Fetch-Dest":            "document",
	"Sec-Fetch-Mode":            "navigate",
	"Sec-Fetch-Site":            "none",
	"Sec-Fetch-User":            "?1",
	"Upgrade-Insecure-Requests": "1",
}

var (
	hdURLPattern = regexp.MustCompile(
		`"progressive_url"\s*:\s*"([^"\\]*(?:\\.[^"\\]*)*)"\s*,\s*"failure_reason"\s*:\s*[^,]+\s*,\s*"metadata"\s*:\s*\{\s*"quality"\s*:\s*"HD"\s*\}`,
	)
	sdURLPattern = regexp.MustCompile(
		`"progressive_url"\s*:\s*"([^"\\]*(?:\\.[^"\\]*)*)"\s*,\s*"failure_reason"\s*:\s*[^,]+\s*,\s*"metadata"\s*:\s*\{\s*"quality"\s*:\s*"SD"\s*\}`,
	)
	titlePattern = regexp.MustCompile(
		`"title"\s*:\s*\{\s*"text"\s*:\s*"([^"\\]*(?:\\.[^"\\]*)*)"`,
	)
)

func GetVideoData(ctx *models.ExtractorContext) (*VideoData, error) {
	contentURL := strings.Replace(ctx.ContentURL, "m.facebook.com", "www.facebook.com", 1)
	contentURL = strings.Replace(contentURL, "mbasic.facebook.com", "www.facebook.com", 1)

	// convert watch URLs to reel permalink,
	// /watch/?v=XXX pages return wrong video data when scraped
	if strings.Contains(contentURL, "/watch") && ctx.ContentID != "" {
		contentURL = "https://www.facebook.com/reel/" + ctx.ContentID
	}

	resp, err := ctx.Fetch(
		http.MethodGet,
		contentURL,
		&networking.RequestParams{
			Headers: webHeaders,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	logger.WriteFile("fb_response", resp)

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get page: %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	return parseVideoFromBody(body, ctx.ContentID)
}

func parseVideoFromBody(body []byte, videoID string) (*VideoData, error) {
	data := &VideoData{}

	// find the section belonging to the requested video
	section := findVideoSection(body, videoID)
	if section == nil {
		// fall back to full body for reel/post pages with a single video
		section = body
	}

	if match := hdURLPattern.FindSubmatch(section); len(match) >= 2 {
		data.HDURL = unescapeFacebookURL(string(match[1]))
	}
	if match := sdURLPattern.FindSubmatch(section); len(match) >= 2 {
		data.SDURL = unescapeFacebookURL(string(match[1]))
	}
	// title can be anywhere in the page
	if match := titlePattern.FindSubmatch(body); len(match) >= 2 {
		data.Title = unescapeUnicode(string(match[1]))
	}

	if data.HDURL == "" && data.SDURL == "" {
		return nil, fmt.Errorf("no video URLs found in page")
	}

	return data, nil
}

// findVideoSection returns the slice of body containing the video delivery
// data for the given videoID, anchored by dash_mpd_debug.mpd?v=VIDEO_ID
// and bounded by the closing "id":"VIDEO_ID".
func findVideoSection(body []byte, videoID string) []byte {
	if videoID == "" {
		return nil
	}

	anchor := []byte("dash_mpd_debug.mpd?v=" + videoID)
	start := bytes.Index(body, anchor)
	if start == -1 {
		return nil
	}

	remaining := body[start:]

	// look for "id":"VIDEO_ID" which closes the videoDeliveryResponseResult block
	endMarker := []byte(`"id":"` + videoID + `"`)
	endIdx := bytes.Index(remaining, endMarker)
	if endIdx > 0 {
		return remaining[:endIdx+len(endMarker)]
	}

	// fallback: take a generous window
	maxLen := 20000
	if maxLen > len(remaining) {
		maxLen = len(remaining)
	}
	return remaining[:maxLen]
}

func unescapeFacebookURL(s string) string {
	s = strings.ReplaceAll(s, `\/`, "/")
	s = unescapeUnicode(s)
	return s
}

func unescapeUnicode(s string) string {
	var b strings.Builder
	b.Grow(len(s))

	for i := 0; i < len(s); {
		if i+5 < len(s) && s[i] == '\\' && s[i+1] == 'u' {
			var r rune
			valid := true
			for j := 2; j < 6; j++ {
				r <<= 4
				c := s[i+j]
				switch {
				case c >= '0' && c <= '9':
					r |= rune(c - '0')
				case c >= 'a' && c <= 'f':
					r |= rune(c - 'a' + 10)
				case c >= 'A' && c <= 'F':
					r |= rune(c - 'A' + 10)
				default:
					valid = false
				}
			}
			if valid && utf8.ValidRune(r) {
				b.WriteRune(r)
				i += 6
				continue
			}
		}
		b.WriteByte(s[i])
		i++
	}
	return b.String()
}
