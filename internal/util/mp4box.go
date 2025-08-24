package util

import (
	"os"

	"github.com/abema/go-mp4"
	"github.com/sunfish-shogi/bufseekio"
)

func ExtractMP4Metadata(file string) (int64, int64, int64) {
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
		seconds := int64(track.Duration / uint64(track.Timescale))
		width := int64(track.AVC.Width)
		height := int64(track.AVC.Height)
		return seconds, width, height
	}
	return 0, 0, 0
}
