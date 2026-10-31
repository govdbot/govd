package facebook

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/govdbot/govd/internal/config"
	"github.com/govdbot/govd/internal/database"
	"github.com/govdbot/govd/internal/models"
	"github.com/govdbot/govd/internal/util"
)

type helperOutput struct {
	bytes.Buffer
	remaining int
}

func (b *helperOutput) Write(p []byte) (int, error) {
	if len(p) > b.remaining {
		return 0, fmt.Errorf("extractor output exceeds limit")
	}
	n, e := b.Buffer.Write(p)
	b.remaining -= n
	return n, e
}

type publicVideo struct {
	ID          string         `json:"id"`
	URL         string         `json:"url"`
	Description string         `json:"description"`
	Title       string         `json:"title"`
	Duration    float64        `json:"duration"`
	FileSize    int64          `json:"filesize"`
	IsLive      bool           `json:"is_live"`
	LiveStatus  string         `json:"live_status"`
	Entries     []*publicVideo `json:"entries"`
}

var publicID = regexp.MustCompile(`^[A-Za-z0-9]{1,100}$`)

func publicFacebookURL(raw string) bool {
	u, e := url.Parse(raw)
	if e != nil || u.Scheme != "https" || u.User != nil || (u.Port() != "" && u.Port() != "443") {
		return false
	}
	switch strings.ToLower(u.Hostname()) {
	case "facebook.com", "www.facebook.com", "m.facebook.com", "mbasic.facebook.com":
		for _, p := range []*regexp.Regexp{videoURLPattern, shareURLPattern} {
			if idx := p.FindStringIndex(raw); idx != nil && idx[0] == 0 {
				return true
			}
		}
		return false
	}
	return false
}
func publicFacebookMediaURL(raw string) bool {
	u, e := url.Parse(raw)
	if e != nil || u.Scheme != "https" || u.User != nil || (u.Port() != "" && u.Port() != "443") {
		return false
	}
	h := strings.ToLower(u.Hostname())
	return h == "fbcdn.net" || strings.HasSuffix(h, ".fbcdn.net")
}
func parsePublicVideo(ctx *models.ExtractorContext, info *publicVideo) (*models.Media, error) {
	if info == nil {
		return nil, util.ErrUnavailable
	}
	if len(info.Entries) > 0 {
		if len(info.Entries) != 1 || info.Entries[0] == nil {
			return nil, util.ErrUnavailable
		}
		info = info.Entries[0]
	}
	if !publicID.MatchString(info.ID) || info.IsLive || info.LiveStatus == "is_live" || !publicFacebookMediaURL(info.URL) {
		return nil, util.ErrUnavailable
	}
	if math.IsNaN(info.Duration) || math.IsInf(info.Duration, 0) || info.Duration < 0 || info.FileSize < 0 {
		return nil, util.ErrUnavailable
	}
	if _, e := strconv.ParseUint(ctx.ContentID, 10, 64); e == nil && !strings.Contains(ctx.ContentURL, "/posts/") && !strings.Contains(ctx.ContentURL, "/post/") && info.ID != ctx.ContentID {
		return nil, util.ErrUnavailable
	}
	if info.Duration > config.Env.MaxDuration.Seconds() {
		return nil, util.ErrDurationTooLong
	}
	if info.FileSize > config.Env.MaxFileSize {
		return nil, util.ErrFileTooLarge
	}
	m := ctx.NewMedia()
	m.SetCaption(info.Description)
	if m.Caption == "" {
		m.SetCaption(info.Title)
	}
	m.NewItem().AddFormats(&models.MediaFormat{Type: database.MediaTypeVideo, FormatID: "public", URL: []string{info.URL}, Duration: int32(math.Ceil(info.Duration)), FileSize: info.FileSize, VideoCodec: database.MediaCodecAvc, AudioCodec: database.MediaCodecAac})
	return m, nil
}
func GetPublicMedia(ctx *models.ExtractorContext) (*models.Media, error) {
	if !publicFacebookURL(ctx.ContentURL) {
		return nil, util.ErrUnavailable
	}
	binary, e := exec.LookPath("yt-dlp")
	if e != nil {
		return nil, fmt.Errorf("optional yt-dlp helper is not installed")
	}
	parent := ctx.Context
	if parent == nil {
		parent = context.Background()
	}
	deadline, cancel := context.WithTimeout(parent, 60*time.Second)
	defer cancel()
	command := exec.CommandContext(deadline, binary, "--ignore-config", "--no-plugin-dirs", "--no-cache-dir", "--no-playlist", "--skip-download", "--dump-single-json", "--socket-timeout", "12", "--retries", "1", "--extractor-retries", "1", "--impersonate", "chrome", "--format", "best[ext=mp4][vcodec!=none][acodec!=none]/best[ext=mp4]", "--", ctx.ContentURL)
	command.Env = []string{}
	for _, name := range []string{"PATH", "LANG", "SSL_CERT_FILE", "SSL_CERT_DIR", "TMPDIR"} {
		if v, ok := os.LookupEnv(name); ok {
			command.Env = append(command.Env, name+"="+v)
		}
	}
	stdout := &helperOutput{remaining: 2 << 20}
	stderr := &helperOutput{remaining: 16 << 10}
	command.Stdout = stdout
	command.Stderr = stderr
	if e := command.Run(); e != nil {
		if deadline.Err() != nil {
			return nil, util.ErrTimeout
		}
		return nil, fmt.Errorf("anonymous Facebook extraction failed: %w", e)
	}
	var info publicVideo
	if e := json.Unmarshal(stdout.Bytes(), &info); e != nil {
		return nil, util.ErrUnavailable
	}
	m, e := parsePublicVideo(ctx, &info)
	if e != nil {
		return nil, e
	}
	client := *ctx.HTTPClient
	client.Cookies = nil
	client.Headers = nil
	if standard, ok := client.Client.(*http.Client); ok {
		copy := *standard
		copy.Jar = nil
		client.Client = &copy
	}
	ctx.HTTPClient = &client
	return m, nil
}
