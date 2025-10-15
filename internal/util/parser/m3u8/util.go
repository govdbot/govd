package m3u8

import (
	"net/url"
	"strconv"
	"strings"

	"github.com/grafov/m3u8"
)

func (p *M3U8Parser) resolveURL(uri string) string {
	if strings.HasPrefix(uri, "http://") || strings.HasPrefix(uri, "https://") {
		return uri
	}
	ref, err := url.Parse(uri)
	if err != nil {
		return uri
	}
	return p.BaseURL.ResolveReference(ref).String()
}

func parseResolution(resolution string) (int32, int32) {
	if resolution == "" {
		return 0, 0
	}
	parts := strings.Split(resolution, "x")
	if len(parts) != 2 {
		return 0, 0
	}
	width, _ := strconv.ParseInt(parts[0], 10, 32)
	height, _ := strconv.ParseInt(parts[1], 10, 32)
	return int32(width), int32(height)
}

func getValidVariants(variants []*m3u8.Variant) []*m3u8.Variant {
	valid := make([]*m3u8.Variant, 0, len(variants))
	for _, variant := range variants {
		if variant != nil && variant.URI != "" {
			valid = append(valid, variant)
		}
	}
	return valid
}
