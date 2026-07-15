package facebook

import (
	"bytes"
	"fmt"
	"html"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"
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
	// These are tried in order; the longest non-empty match wins.
	ogDescPattern = regexp.MustCompile(
		`<meta\s+property="og:description"\s+content="([^"]*)"\s*/?>`,
	)
	ogTitlePattern = regexp.MustCompile(
		`<meta\s+property="og:title"\s+content="([^"]*)"\s*/?>`,
	)
	messagePattern = regexp.MustCompile(
		`"(?:message|attached_story)"\s*:\s*\{\s*"text"\s*:\s*"([^"\\]*(?:\\.[^"\\]*)*)"`,
	)
	storyPattern = regexp.MustCompile(
		`"story"\s*:\s*\{\s*"comet_sections"\s*:\s*\{[^}]*"message"\s*:\s*\{\s*"text"\s*:\s*"([^"\\]*(?:\\.[^"\\]*)*)"`,
	)
	// creation_story.comet_sections – another FB wrapper (reels posted via page)
	creationStoryPattern = regexp.MustCompile(
		`"creation_story"\s*:\s*\{[^}]{0,500}"message"\s*:\s*\{\s*"text"\s*:\s*"([^"\\]*(?:\\.[^"\\]*)*)"`,
	)
	// "subtitle" / "description" plain text blocks that sometimes hold the caption
	descriptionPattern = regexp.MustCompile(
		`"description"\s*:\s*\{\s*"text"\s*:\s*"([^"\\]{20,}[^"\\]*(?:\\.[^"\\]*)*)"`,
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

	// Facebook serves variable HTML: sometimes a full video page (with
	// progressive_url + caption), sometimes a light/blocked variant with
	// neither, sometimes HTTP 400/429 anti-bot. A follow-up request frequently
	// lands on a different edge and returns the complete page, so retry the
	// fetch with a short backoff before giving up. Using `impersonate: true`
	// in private/config.yaml (chrome TLS) is REQUIRED to avoid the 400.
	const maxAttempts = 4
	var lastErr error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		if attempt > 1 {
			sleepMS := time.Duration(300*(1<<(attempt-2))) * time.Millisecond
			if sleepMS > 3*time.Second {
				sleepMS = 3 * time.Second
			}
			time.Sleep(sleepMS)
		}

		resp, err := ctx.Fetch(
			http.MethodGet,
			contentURL,
			&networking.RequestParams{
				Headers: webHeaders,
			},
		)
		if err != nil {
			lastErr = fmt.Errorf("failed to send request: %w", err)
			continue
		}

		logger.WriteFile("fb_response", resp)

		if resp.StatusCode != http.StatusOK {
			// FB anti-bot often returns 400/429 on this VPS; treat as retryable
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			if resp.StatusCode == http.StatusBadRequest || resp.StatusCode == 429 || resp.StatusCode >= 500 {
				lastErr = fmt.Errorf("failed to get page: %s (anti-bot, retryable)", resp.Status)
				// keep body snippet for debug if small
				if len(body) > 0 && len(body) < 500 {
					lastErr = fmt.Errorf("%w body=%q", lastErr, string(body))
				}
				continue
			}
			lastErr = fmt.Errorf("failed to get page: %s", resp.Status)
			continue
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			lastErr = fmt.Errorf("failed to read response body: %w", err)
			continue
		}

		// If FB returns a very small / login / checkpoint page (< 5KB), treat as blocked variant
		if len(body) < 5000 {
			// quick check for known blocked markers before parsing
			lb := string(body)
			if strings.Contains(lb, "login") && strings.Contains(lb, "checkpoint") ||
				strings.Contains(lb, "temporarily blocked") ||
				strings.Contains(lb, "you have been temporarily blocked") {
				lastErr = fmt.Errorf("no video URLs found in page (blocked/login page, len=%d)", len(body))
				continue
			}
		}

		data, err := parseVideoFromBody(body, ctx.ContentID)
		if err != nil {
			// Only retry on the "no video URLs" variant (light/blocked
			// page). Other errors (real parse failures) should surface.
			if strings.Contains(err.Error(), "no video URLs found") {
				lastErr = err
				continue
			}
			return nil, err
		}
		// Even if video URLs found but caption is empty, do NOT fail —
		// video itself is valid, caption is best-effort.
		_ = body
		return data, nil
	}

	if lastErr == nil {
		lastErr = fmt.Errorf("no video URLs found in page")
	}
	return nil, lastErr
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

	// findVideoSection anchors on dash_mpd_debug.mpd?v=VIDEO_ID and bounds
	// the block by the following "id":"VIDEO_ID" marker. On some page
	// variants that marker lands BEFORE the progressive_url's trailing
	// failure_reason/metadata (which carries the HD/SD quality), so the
	// URL regexes above cannot match inside the truncated slice and we
	// end up with no URLs even though the page clearly has them. Retry
	// on the full body (single-video posts/reels) so the extraction does
	// not falsely fail. Caption extraction already runs against body.
	if data.HDURL == "" && data.SDURL == "" {
		if match := hdURLPattern.FindSubmatch(body); len(match) >= 2 {
			data.HDURL = unescapeFacebookURL(string(match[1]))
		}
		if match := sdURLPattern.FindSubmatch(body); len(match) >= 2 {
			data.SDURL = unescapeFacebookURL(string(match[1]))
		}
	}
	// title can be anywhere in the page
	if match := titlePattern.FindSubmatch(body); len(match) >= 2 {
		data.Title = unescapeUnicode(string(match[1]))
	}

	// Facebook stores the post caption / description in several places
	// depending on page variant. Collect all candidates and keep the
	// longest one that actually looks like a caption (not a short token).
	// PRIORITY: caption anchored to this videoID (message closest to "id":"videoID")
	candidates := []string{}
	anchoredCaption := findCaptionAnchoredToID(body, videoID)

	// helper: decode + html-unescape + trim + basic sanity (FB often wraps caption in entities)
	addCandidate := func(raw string) {
		s := unescapeUnicode(raw)
		s = html.UnescapeString(s)
		s = strings.TrimSpace(s)
		if s == "" {
			return
		}
		// discard very short tokens that look like UI labels, not captions
		if len(s) < 3 {
			return
		}
		// FB sometimes repeats "Facebook" / "Log in" as og:description fallback – skip
		lower := strings.ToLower(s)
		if lower == "facebook" || strings.HasPrefix(lower, "log in to facebook") || strings.HasPrefix(lower, "log into facebook") {
			return
		}
		s = cleanFacebookCaption(s)
		if s == "" {
			return
		}
		candidates = append(candidates, s)
	}

	// If we have an anchored caption (message directly tied to this video ID), it wins outright
	if anchoredCaption != "" {
		// decode it through same pipeline
		anchoredCaption = strings.TrimSpace(html.UnescapeString(unescapeUnicode(anchoredCaption)))
		if len(anchoredCaption) >= 3 && !strings.EqualFold(anchoredCaption, "facebook") {
			anchoredCaption = cleanFacebookCaption(anchoredCaption)
			if anchoredCaption != "" {
				data.Title = anchoredCaption
				if data.HDURL != "" || data.SDURL != "" {
					return data, nil
				}
			}
		}
	}

	// Only run other regexes if anchored did not already succeed with URLs.
	// This saves ~6 regex scans (60MB) on the happy path.
	if data.Title == "" || data.HDURL == "" && data.SDURL == "" {
		// lightly: try cheap patterns first (title already tried, so og tags)
		if match := ogDescPattern.FindSubmatch(body); len(match) >= 2 {
			addCandidate(string(match[1]))
		}
		if len(candidates) == 0 { // avoid extra scans if we already have a candidate
			if match := ogTitlePattern.FindSubmatch(body); len(match) >= 2 {
				addCandidate(string(match[1]))
			}
		}
		if match := messagePattern.FindSubmatch(body); len(match) >= 2 {
			addCandidate(string(match[1]))
		}
		if len(candidates) == 0 {
			if match := storyPattern.FindSubmatch(body); len(match) >= 2 {
				addCandidate(string(match[1]))
			}
			if match := creationStoryPattern.FindSubmatch(body); len(match) >= 2 {
				addCandidate(string(match[1]))
			}
			if match := descriptionPattern.FindSubmatch(body); len(match) >= 2 {
				addCandidate(string(match[1]))
			}
		}
		// Also scrape ALL "message":{"text":"..."} occurrences and keep longest (page may have multiple)
		// Only if we still need a caption - this is the heaviest scan (FindAllSubmatch)
		if len(candidates) == 0 {
			allMessages := messagePattern.FindAllSubmatch(body, 10)
			for _, m := range allMessages {
				if len(m) >= 2 {
					addCandidate(string(m[1]))
				}
			}
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

	if data.HDURL == "" && data.SDURL == "" {
		return nil, fmt.Errorf("no video URLs found in page")
	}

	return data, nil
}

// findCaptionAnchoredToID looks for the caption belonging to a specific videoID.
// FB reel pages contain multiple reels in the feed; each reel's JSON block looks like:
// {"message":{"text":"..."},"id":"<VIDEO_ID>"}. We search backwards from each occurrence
// of "id":"VIDEO_ID" for the nearest message text within 8KB.
// This prevents picking a longer caption from a different reel in the same page.
func findCaptionAnchoredToID(body []byte, videoID string) string {
	if videoID == "" {
		return ""
	}
	idMarker := []byte(`"id":"` + videoID + `"`)
	// Find all occurrences of the id marker
	for offset := 0; ; {
		idx := bytes.Index(body[offset:], idMarker)
		if idx == -1 {
			break
		}
		absIdx := offset + idx
		// Look backwards up to 8000 bytes for message pattern
		start := absIdx - 8000
		if start < 0 {
			start = 0
		}
		window := body[start:absIdx]
		// The message should be relatively close before the id
		if match := messagePattern.FindSubmatch(window); len(match) >= 2 {
			// There might be multiple messages in window; take the last one (closest to id)
			all := messagePattern.FindAllSubmatch(window, -1)
			if len(all) > 0 {
				last := all[len(all)-1]
				if len(last) >= 2 {
					candidate := string(last[1])
					// Basic sanity: not empty, not just hashtags spam check is handled by addCandidate
					if len(candidate) >= 3 {
						return candidate
					}
				}
			}
		}
		offset = absIdx + len(idMarker)
	}
	return ""
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

// cleanFacebookCaption mimics IG/Threads clean caption approach.
// FB often returns "PageName\n\nActual caption" in message/creation_story fields,
// where PageName is like "General mbappe" (short, no hashtags). IG uses dedicated
// EdgeMediaToCaption field that is pure caption; we strip the prefix for FB to match.
func cleanFacebookCaption(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	// Normalize: \r\n -> \n
	s = strings.ReplaceAll(s, "\r\n", "\n")
	// Split by double newline – FB separates page name and caption with \n\n
	parts := strings.Split(s, "\n\n")
	if len(parts) >= 2 {
		first := strings.TrimSpace(parts[0])
		// If first part itself contains single newline, e.g. "General mbappe\nRandom memes..."
		// then first line of it may be the page name
		if strings.Contains(first, "\n") {
			subParts := strings.Split(first, "\n")
			subFirst := strings.TrimSpace(subParts[0])
			subRest := strings.TrimSpace(strings.Join(subParts[1:], "\n"))
			restAll := subRest
			if len(parts) > 1 {
				restAll = restAll + "\n\n" + strings.TrimSpace(strings.Join(parts[1:], "\n\n"))
			}
			restAll = strings.TrimSpace(restAll)
			if subFirst != "" && restAll != "" && looksLikeFBPageName(subFirst) && len(restAll) > len(subFirst) {
				return restAll
			}
		}
		rest := strings.TrimSpace(strings.Join(parts[1:], "\n\n"))
		if first == "" || rest == "" {
			// fall through
		} else if looksLikeFBPageName(first) && len(rest) > len(first) {
			return rest
		}
		return s
	}
	// Fallback: try single newline split if first line looks like page name
	// e.g. "General mbappe\nRandom memes from my phone..."
	lines := strings.Split(s, "\n")
	if len(lines) >= 2 {
		first := strings.TrimSpace(lines[0])
		rest := strings.TrimSpace(strings.Join(lines[1:], "\n"))
		if first != "" && rest != "" && looksLikeFBPageName(first) && len(rest) > len(first) {
			return rest
		}
	}
	return s
}

func looksLikeFBPageName(s string) bool {
	s = strings.TrimSpace(s)
	if s == "" {
		return false
	}
	// Page names are typically short, no hashtags, no URLs, no emoji-heavy
	if len(s) > 60 {
		return false
	}
	if strings.Contains(s, "#") {
		return false
	}
	if strings.Contains(s, "http://") || strings.Contains(s, "https://") {
		return false
	}
	// If it looks like a sentence with many words but no hashtag, still could be caption
	// Heuristic: if it contains newline, not page name
	if strings.Contains(s, "\n") {
		return false
	}
	// If it has many words (e.g. full sentence), treat as part of caption
	// Page category like "General mbappe" is 2 words, short
	// But "Random memes from my phone pt.1011" is longer and IS caption – keep it
	// So we check: if first part is <= 3 words and second part exists, it's likely page name
	words := strings.Fields(s)
	if len(words) <= 4 {
		// Check if it looks like a category/name (e.g. "General mbappe", "Memes", "Funny videos")
		// No punctuation heavy, not a full sentence
		if len(s) <= 40 {
			return true
		}
	}
	// Additional heuristic: common FB junk prefixes that appear alone
	lower := strings.ToLower(s)
	junkPrefixes := []string{"general", "meme", "funny", "reels", "viral"}
	// If first part is exactly a short category-like word, treat as page name
	// e.g. "General mbappe" -> contains "general" + short
	for _, jp := range junkPrefixes {
		if strings.HasPrefix(lower, jp) && len(s) <= 30 {
			return true
		}
	}
	return false
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
