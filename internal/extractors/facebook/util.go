package facebook

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf16"
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

// videoFetchHeaders mirrors yt-dlp's behaviour: Facebook CDN
// rate-limits browser User-Agents, so use a non-browser UA when we
// hit the progressive video URL (HEAD or GET).
var videoFetchHeaders = map[string]string{
	"User-Agent": "facebookexternalhit/1.1",
	"Accept":     "*/*",
	"Range":      "bytes=0-0",
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
	// FB modern pages embed the post caption in several places.
	// These are tried in order; the first non-empty, longest match wins.
	ogDescPattern = regexp.MustCompile(
		`<meta\s+property="og:description"\s+content="([^"]*)"\s*/?>`,
	)
	messagePattern = regexp.MustCompile(
		`"(?:message|attached_story)"\s*:\s*\{\s*"text"\s*:\s*"([^"\\]*(?:\\.[^"\\]*)*)"`,
	)
	storyPattern = regexp.MustCompile(
		`"story"\s*:\s*\{\s*"comet_sections"\s*:\s*\{[^}]*"message"\s*:\s*\{\s*"text"\s*:\s*"([^"\\]*(?:\\.[^"\\]*)*)"`,
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

	// Facebook stores the post caption / description in several places
	// depending on page variant. Collect all candidates and keep the
	// longest one that actually looks like a caption (not a short token).
	candidates := []string{}
	if match := ogDescPattern.FindSubmatch(body); len(match) >= 2 {
		if c := strings.TrimSpace(unescapeUnicode(string(match[1]))); c != "" {
			candidates = append(candidates, c)
		}
	}
	if match := messagePattern.FindSubmatch(body); len(match) >= 2 {
		if c := strings.TrimSpace(unescapeUnicode(string(match[1]))); c != "" {
			candidates = append(candidates, c)
		}
	}
	if match := storyPattern.FindSubmatch(body); len(match) >= 2 {
		if c := strings.TrimSpace(unescapeUnicode(string(match[1]))); c != "" {
			candidates = append(candidates, c)
		}
	}

	best := data.Title
	for _, c := range candidates {
		// prefer the longest non-trivial caption
		if len(c) > len(best) {
			best = c
		}
	}
	data.Title = best

	// DEBUG: dump the exact raw caption bytes Facebook returned so we can
	// see how emoji are encoded. Writes directly (bypasses logger level).
	_ = os.WriteFile("/app/logs/fb_caption_debug.txt",
		[]byte(fmt.Sprintf("raw=%q\nunescaped=%q", best, unescapeUnicode(best))), 0644)

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

// probeFormatSize asks the Facebook CDN for the size of a progressive
// video URL. We send a HEAD first; some FBCDN edges reject HEAD with
// 405, so we fall back to a single-byte ranged GET. Any unrecoverable
// error returns 0 -- callers must treat that as "size unknown" and
// fall back to a different format.
func probeFormatSize(ctx *models.ExtractorContext, url string) int64 {
	if url == "" {
		return 0
	}

	if size := probeContentLength(ctx, http.MethodHead, url, nil); size > 0 {
		return size
	}

	rangeHeaders := map[string]string{
		"User-Agent": videoFetchHeaders["User-Agent"],
		"Accept":     "*/*",
		"Range":      "bytes=0-0",
	}
	return probeContentLength(ctx, http.MethodGet, url, rangeHeaders)
}

func probeContentLength(
	ctx *models.ExtractorContext,
	method string,
	url string,
	headers map[string]string,
) int64 {
	if headers == nil {
		headers = map[string]string{
			"User-Agent": videoFetchHeaders["User-Agent"],
			"Accept":     "*/*",
		}
	}
	resp, err := ctx.Fetch(
		method,
		url,
		&networking.RequestParams{Headers: headers},
	)
	if err != nil {
		return 0
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusPartialContent {
		if total := parseContentRangeTotal(resp.Header.Get("Content-Range")); total > 0 {
			return total
		}
	}

	if resp.StatusCode >= 200 && resp.StatusCode < 300 && resp.ContentLength > 0 {
		return resp.ContentLength
	}
	return 0
}

func parseContentRangeTotal(header string) int64 {
	header = strings.TrimSpace(header)
	if header == "" {
		return 0
	}
	idx := strings.LastIndex(header, "/")
	if idx == -1 {
		return 0
	}
	totalPart := strings.TrimSpace(header[idx+1:])
	if totalPart == "" || totalPart == "*" {
		return 0
	}
	var total int64
	for _, c := range totalPart {
		if c < '0' || c > '9' {
			return 0
		}
		total = total*10 + int64(c-'0')
	}
	return total
}

func unescapeUnicode(s string) string {
	// Facebook embeds JSON inside HTML, so backslashes are themselves
	// escaped: "\uD83E\uDD40" arrives as "\\uD83E\\uDD40". Normalise the
	// double backslash form back to a single one before decoding.
	s = strings.ReplaceAll(s, `\\u`, `\u`)
	s = strings.ReplaceAll(s, `\\/`, `/`)

	var b strings.Builder
	b.Grow(len(s))

	for i := 0; i < len(s); {
		if i+5 < len(s) && s[i] == '\\' && s[i+1] == 'u' {
			r, ok := decodeHex4(s[i+2 : i+6])
			if ok {
				// High surrogate? Look ahead for a following \uXXXX low surrogate.
				if utf16.IsSurrogate(r) && i+11 < len(s) &&
					s[i+6] == '\\' && s[i+7] == 'u' {
					if r2, ok2 := decodeHex4(s[i+8 : i+12]); ok2 {
						if dec := utf16.DecodeRune(r, r2); dec != unicode.ReplacementChar {
							b.WriteRune(dec)
							i += 12
							continue
						}
					}
				}
				if utf8.ValidRune(r) {
					b.WriteRune(r)
					i += 6
					continue
				}
			}
		}
		b.WriteByte(s[i])
		i++
	}
	return b.String()
}

// decodeHex4 parses a 4-char hex sequence (e.g. "D83E") into a rune.
func decodeHex4(h string) (rune, bool) {
	var r rune
	for j := 0; j < 4; j++ {
		r <<= 4
		c := h[j]
		switch {
		case c >= '0' && c <= '9':
			r |= rune(c - '0')
		case c >= 'a' && c <= 'f':
			r |= rune(c - 'a' + 10)
		case c >= 'A' && c <= 'F':
			r |= rune(c - 'A' + 10)
		default:
			return 0, false
		}
	}
	return r, true
}
