package m3u8

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/govdbot/govd/internal/models"
	"github.com/govdbot/govd/internal/networking"
	"github.com/grafov/m3u8"
)

const MAX_CONCURRENT_REQUESTS = 5

type M3U8Parser struct {
	Context       *models.ExtractorContext
	BaseURL       *url.URL
	Playlist      m3u8.Playlist
	PlaylistType  m3u8.ListType
	RequestParams *networking.RequestParams
}

func ParseM3U8(
	ctx *models.ExtractorContext,
	baseURL string,
	data []byte,
	requestParams *networking.RequestParams,
) ([]*models.MediaFormat, error) {
	baseURLObj, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("invalid base URL %q: %w", baseURL, err)
	}

	buf := bytes.NewBuffer(data)
	playlist, listType, err := m3u8.DecodeFrom(buf, false)
	if err != nil {
		return nil, fmt.Errorf("failed parsing M3U8: %w", err)
	}

	parser := &M3U8Parser{
		Context:       ctx,
		BaseURL:       baseURLObj,
		Playlist:      playlist,
		PlaylistType:  listType,
		RequestParams: requestParams,
	}

	return parser.Parse()
}

func ParseM3U8FromURL(
	ctx *models.ExtractorContext,
	url string,
	requestParams *networking.RequestParams,
) ([]*models.MediaFormat, error) {
	resp, err := ctx.Fetch(
		http.MethodGet,
		url, requestParams,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch M3U8 playlist: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("failed to fetch M3U8 playlist, status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read M3U8 playlist body: %w", err)
	}

	return ParseM3U8(ctx, resp.Request.URL.String(), body, requestParams)
}

func (p *M3U8Parser) Parse() ([]*models.MediaFormat, error) {
	switch p.PlaylistType {
	case m3u8.MASTER:
		master, ok := p.Playlist.(*m3u8.MasterPlaylist)
		if !ok {
			return nil, fmt.Errorf("failed to cast to master playlist")
		}
		return p.parseMasterPlaylist(master)
	case m3u8.MEDIA:
		media, ok := p.Playlist.(*m3u8.MediaPlaylist)
		if !ok {
			return nil, fmt.Errorf("failed to cast to media playlist")
		}
		return p.parseMediaPlaylist(media)
	default:
		return nil, fmt.Errorf("unsupported M3U8 playlist type: %v", p.PlaylistType)
	}
}
