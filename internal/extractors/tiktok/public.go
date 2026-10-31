package tiktok

import (
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"github.com/govdbot/govd/internal/models"
	"github.com/govdbot/govd/internal/networking"
	"github.com/govdbot/govd/internal/util"
)

const mobileUserAgent = "Mozilla/5.0 (Linux; Android 14) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/130.0.0.0 Mobile Safari/537.36"

var videoIDPattern = regexp.MustCompile(`^[0-9]{8,25}$`)
var frontityPattern = regexp.MustCompile(`(?s)<script[^>]+\bid=["']__FRONTITY_CONNECT_STATE__["'][^>]*>(.*?)</script>`)

func publicTikTokURL(raw string) bool {
	u, e := url.Parse(raw)
	if e != nil || u.Scheme != "https" || u.User != nil || (u.Port() != "" && u.Port() != "443") {
		return false
	}
	h := strings.ToLower(u.Hostname())
	for _, d := range []string{"tiktok.com", "tiktokv.com", "tiktokv.eu", "tiktokv.us", "tiktokcdn.com", "tiktokcdn-eu.com", "tiktokcdn-us.com", "tiktokeu-cdn.com", "byteoversea.com", "ibytedtos.com", "ibyteimg.com", "muscdn.com", "byteicdn.com"} {
		if h == d || strings.HasSuffix(h, "."+d) {
			return true
		}
	}
	return false
}
func validTikTokURLs(raw []string) []string {
	var result []string
	for _, u := range raw {
		if len(result) >= 8 {
			break
		}
		if publicTikTokURL(u) {
			result = append(result, u)
		}
	}
	return result
}
func publicTikTokHeaders() map[string]string {
	return map[string]string{"User-Agent": mobileUserAgent, "Referer": "https://www.tiktok.com/", "Accept-Language": "en-US,en;q=0.9"}
}

// Preserve only the anonymous site's cookie scopes; never turn a host-only
// cookie into a cookie for unrelated video CDN domains.
func publicTikTokCookies(cookies []*http.Cookie) []*http.Cookie {
	var result []*http.Cookie
	for _, cookie := range cookies {
		if cookie == nil {
			continue
		}
		copy := *cookie
		if copy.Domain == "" {
			copy.Domain = "www.tiktok.com"
		}
		domain := strings.TrimPrefix(strings.ToLower(copy.Domain), ".")
		if domain != "tiktok.com" && !strings.HasSuffix(domain, ".tiktok.com") {
			continue
		}
		result = append(result, &copy)
	}
	return result
}

func GetVideoEmbed(ctx *models.ExtractorContext) (*WebItemStruct, error) {
	if !videoIDPattern.MatchString(ctx.ContentID) {
		return nil, util.ErrUnavailable
	}
	resp, e := ctx.Fetch(http.MethodGet, "https://www.tiktok.com/embed/v2/"+ctx.ContentID, &networking.RequestParams{Headers: publicTikTokHeaders()})
	if e != nil {
		return nil, e
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, util.ErrUnavailable
	}
	body, e := io.ReadAll(resp.Body)
	if e != nil {
		return nil, e
	}
	return parseVideoEmbed(body, ctx.ContentID)
}
func parseVideoEmbed(body []byte, id string) (*WebItemStruct, error) {
	if !videoIDPattern.MatchString(id) {
		return nil, util.ErrUnavailable
	}
	match := frontityPattern.FindSubmatch(body)
	if len(match) < 2 {
		return nil, util.ErrUnavailable
	}
	type itemInfo struct {
		ID        string `json:"id"`
		Text      string `json:"text"`
		Secret    bool   `json:"secret"`
		ForFriend bool   `json:"forFriend"`
		Video     struct {
			URLs []string `json:"urls"`
			Meta struct {
				Duration float64 `json:"duration"`
				Width    int32   `json:"width"`
				Height   int32   `json:"height"`
			} `json:"videoMeta"`
		} `json:"video"`
	}
	type page struct {
		IsError   bool `json:"isError"`
		VideoData struct {
			Item itemInfo `json:"itemInfos"`
		} `json:"videoData"`
	}
	var data struct {
		Source struct {
			Data map[string]json.RawMessage `json:"data"`
		} `json:"source"`
	}
	if e := json.Unmarshal(match[1], &data); e != nil {
		return nil, util.ErrUnavailable
	}
	raw, ok := data.Source.Data["/embed/v2/"+id]
	if !ok {
		raw, ok = data.Source.Data["/embed/"+id]
	}
	var p page
	if !ok || json.Unmarshal(raw, &p) != nil {
		return nil, util.ErrUnavailable
	}
	i := p.VideoData.Item
	if !ok || p.IsError || i.ID != id || i.Secret || i.ForFriend {
		return nil, util.ErrUnavailable
	}
	urls := validTikTokURLs(i.Video.URLs)
	if len(urls) == 0 {
		return nil, util.ErrUnavailable
	}
	duration := i.Video.Meta.Duration
	if duration < 0 || duration > 86400 {
		return nil, util.ErrUnavailable
	}
	return &WebItemStruct{ID: id, Desc: i.Text, Video: &WebVideo{Duration: int32(math.Ceil(duration)), Width: i.Video.Meta.Width, Height: i.Video.Meta.Height, PlayAddr: &WebPlayAddr{URI: id, URLList: urls}}}, nil
}
func ensureItemIdentity(details *WebItemStruct, id string) error {
	if details == nil || !videoIDPattern.MatchString(id) || details.ID != id {
		return fmt.Errorf("TikTok item does not match requested video")
	}
	if details.PrivateItem || details.Secret || details.ForFriend {
		return util.ErrUnavailable
	}
	return nil
}
func hasDownloadableVideo(v *WebVideo) bool {
	if v == nil {
		return false
	}
	if v.PlayAddr != nil && len(validTikTokURLs(v.PlayAddr.URLList)) > 0 {
		return true
	}
	return len(validTikTokURLs([]string{v.PlayURL, v.DownloadURL})) > 0
}
