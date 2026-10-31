package instagram

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"github.com/bytedance/sonic"
	"github.com/govdbot/govd/internal/config"
	"github.com/govdbot/govd/internal/database"
	"github.com/govdbot/govd/internal/models"
	"github.com/govdbot/govd/internal/util"
)

var publicShortcodePattern = regexp.MustCompile(`^[A-Za-z0-9_-]{1,28}$`)

type cappedOutput struct {
	bytes.Buffer
	remaining int
}

func (b *cappedOutput) Write(p []byte) (int, error) {
	if len(p) > b.remaining {
		return 0, fmt.Errorf("extractor output exceeded size limit")
	}
	n, err := b.Buffer.Write(p)
	b.remaining -= n
	return n, err
}

type ytdlpMedia struct {
	ID          string            `json:"id"`
	Thumbnail   string            `json:"thumbnail"`
	Formats     []json.RawMessage `json:"formats"`
	URL         string            `json:"url"`
	Description string            `json:"description"`
	Duration    float64           `json:"duration"`
	FileSize    int64             `json:"filesize"`
	Entries     []*ytdlpMedia     `json:"entries"`
}

func validMediaURL(value string) bool {
	u, err := url.Parse(value)
	return err == nil && u.Hostname() != "" && (u.Scheme == "https" || u.Scheme == "http")
}

func publicInstagramCDN(value string) bool {
	u, err := url.Parse(value)
	if err != nil || u.Scheme != "https" || u.User != nil {
		return false
	}
	host := strings.ToLower(u.Hostname())
	for _, domain := range []string{"cdninstagram.com", "fbcdn.net", "instagram.com"} {
		if host == domain || strings.HasSuffix(host, "."+domain) {
			return true
		}
	}
	return false
}

func parseYTDLPMedia(ctx *models.ExtractorContext, info *ytdlpMedia) (*models.Media, error) {
	if info == nil {
		return nil, util.ErrUnavailable
	}
	media := ctx.NewMedia()
	media.SetCaption(info.Description)
	entries := info.Entries
	if len(entries) == 0 {
		entries = []*ytdlpMedia{info}
	}
	if len(entries) > 30 {
		return nil, util.ErrMediaAlbumGlobalLimitExceeded
	}
	for _, entry := range entries {
		if entry == nil {
			return nil, util.ErrUnavailable
		}
		if entry.URL == "" && len(entry.Formats) == 0 && entry.Duration == 0 && publicInstagramCDN(entry.Thumbnail) && !strings.Contains(ctx.ContentURL, "/reel") {
			media.NewItem().AddFormats(&models.MediaFormat{FormatID: imageFormatID, Type: database.MediaTypePhoto, URL: []string{entry.Thumbnail}})
			continue
		}
		if !publicInstagramCDN(entry.URL) {
			return nil, fmt.Errorf("extractor returned an invalid Instagram media URL")
		}
		if entry.Duration > config.Env.MaxDuration.Seconds() {
			return nil, util.ErrDurationTooLong
		}
		if entry.FileSize > config.Env.MaxFileSize {
			return nil, util.ErrFileTooLarge
		}
		media.NewItem().AddFormats(&models.MediaFormat{FormatID: videoFormatID, Type: database.MediaTypeVideo, URL: []string{entry.URL}, VideoCodec: database.MediaCodecAvc, AudioCodec: database.MediaCodecAac, Duration: int32(math.Ceil(entry.Duration)), FileSize: entry.FileSize})
	}
	return media, nil
}

// Optional local fallback. yt-dlp only extracts metadata; normal govd download
// and delivery still enforce resource limits and clean up temporary files.
func runPublicHelper(ctx *models.ExtractorContext, canonicalURL string) (*ytdlpMedia, error) {
	binary, err := exec.LookPath("yt-dlp")
	if err != nil {
		return nil, fmt.Errorf("optional yt-dlp extractor is not installed")
	}
	parent := ctx.Context
	if parent == nil {
		parent = context.Background()
	}
	deadline, cancel := context.WithTimeout(parent, 60*time.Second)
	defer cancel()
	command := exec.CommandContext(deadline, binary, "--ignore-config", "--no-plugin-dirs", "--no-cache-dir", "--no-playlist", "--skip-download", "--dump-single-json", "--ignore-no-formats-error", "--socket-timeout", "15", "--retries", "1", "--extractor-retries", "1", "--impersonate", "chrome", "--format", "best[ext=mp4][vcodec!=none][acodec!=none]/best[ext=mp4]", "--", canonicalURL)
	// Do not hand the child process BOT_TOKEN, database passwords, browser
	// profiles, or the operator's environment-based configuration.
	for _, name := range []string{"PATH", "LANG", "SSL_CERT_FILE", "SSL_CERT_DIR", "TMPDIR"} {
		if value, ok := os.LookupEnv(name); ok {
			command.Env = append(command.Env, name+"="+value)
		}
	}
	stdout := &cappedOutput{remaining: 2 << 20}
	stderr := &cappedOutput{remaining: 16 << 10}
	command.Stdout = stdout
	command.Stderr = stderr
	if err := command.Run(); err != nil {
		if deadline.Err() != nil {
			return nil, util.ErrTimeout
		}
		return nil, fmt.Errorf("public yt-dlp extraction failed: %w", err)
	}
	var info ytdlpMedia
	if err := sonic.Unmarshal(stdout.Bytes(), &info); err != nil {
		return nil, fmt.Errorf("invalid yt-dlp metadata: %w", err)
	}
	return &info, nil
}

func GetYTDLPMedia(ctx *models.ExtractorContext) (*models.Media, error) {
	if !publicShortcodePattern.MatchString(ctx.ContentID) {
		return nil, fmt.Errorf("invalid Instagram shortcode")
	}
	info, err := runPublicHelper(ctx, "https://www.instagram.com/p/"+ctx.ContentID+"/")
	if err != nil {
		return nil, err
	}
	media, err := parseYTDLPMedia(ctx, info)
	if err != nil {
		return nil, err
	}
	// Public CDN downloads must not inherit any account cookies or headers.
	client := *ctx.HTTPClient
	client.Cookies = nil
	client.Headers = nil
	if standard, ok := client.Client.(*http.Client); ok {
		cloned := *standard
		cloned.Jar = nil
		client.Client = &cloned
	}
	ctx.HTTPClient = &client
	return media, nil
}

func resolvePublicShare(ctx *models.ExtractorContext) (string, error) {
	if !publicShortcodePattern.MatchString(ctx.ContentID) {
		return "", util.ErrUnavailable
	}
	u, err := url.Parse(ctx.ContentURL)
	if err != nil {
		return "", util.ErrUnavailable
	}
	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(parts) < 2 || len(parts) > 3 || parts[0] != "share" {
		return "", util.ErrUnavailable
	}
	path := "/share/"
	if len(parts) == 3 {
		switch parts[1] {
		case "reel", "reels", videoFormatID, "s", "p":
			path += parts[1] + "/"
		default:
			return "", util.ErrUnavailable
		}
	}
	info, err := runPublicHelper(ctx, "https://www.instagram.com"+path+ctx.ContentID+"/")
	if err != nil {
		return "", err
	}
	if !publicShortcodePattern.MatchString(info.ID) {
		return "", util.ErrUnavailable
	}
	return "https://www.instagram.com/p/" + info.ID + "/", nil
}

var _ io.Writer = (*cappedOutput)(nil)
