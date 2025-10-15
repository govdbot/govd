package m3u8

import (
	"fmt"
	"io"
	"net/http"

	"github.com/govdbot/govd/internal/models"
	"github.com/govdbot/govd/internal/util"
	"github.com/grafov/m3u8"
)

func (p *M3U8Parser) handleEncryption(
	playlist *m3u8.MediaPlaylist,
	format *models.MediaFormat,
) error {
	if playlist.Key == nil || playlist.Key.URI == "" {
		return nil
	}
	keyURL := p.resolveURL(playlist.Key.URI)
	resp, err := p.Context.Fetch(
		http.MethodGet,
		keyURL,
		nil,
	)
	if err != nil {
		return fmt.Errorf("failed to fetch encryption key: %w", err)
	}
	defer resp.Body.Close()

	iv, err := util.ParseHex(playlist.Key.IV)
	if err != nil {
		return fmt.Errorf("invalid initialization vector: %w", err)
	}

	key, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read encryption key: %w", err)
	}

	format.DecryptionKey = &models.DecryptionKey{
		Method:        playlist.Key.Method,
		Key:           key,
		IV:            iv,
		MediaSequence: int(playlist.SeqNo),
	}

	return nil
}
