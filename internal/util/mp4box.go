package util

import (
	"os"

	"github.com/abema/go-mp4"
	"github.com/govdbot/govd/internal/logger"
	"github.com/sunfish-shogi/bufseekio"
)

func ExtractMP4Metadata(file string) (int32, int32, int32) {
	logger.L.Debugf("extracting mp4 metadata: %s", file)

	buf, err := os.Open(file)
	if err != nil {
		return 0, 0, 0
	}
	defer buf.Close()

	r := bufseekio.NewReadSeeker(buf, 1024, 4)
	info, err := mp4.Probe(r)
	if err != nil {
		return 0, 0, 0
	}
	for _, track := range info.Tracks {
		if track.AVC == nil {
			continue
		}
		seconds := int32(track.Duration / uint64(track.Timescale))
		width := int32(track.AVC.Width)
		height := int32(track.AVC.Height)
		return seconds, width, height
	}
	return 0, 0, 0
}
