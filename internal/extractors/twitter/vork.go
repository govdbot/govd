package twitter

import (
	"os"
	"strings"

	"github.com/abema/go-mp4"
	"github.com/sunfish-shogi/bufseekio"
)

// vork is an internal muxer used by Twitter and needs
// double muxing to be compatible with standard players.
// learn more: https://github.com/govdbot/govd/issues/29
func IsVorkMuxer(inputPath string) bool {
	buf, err := os.Open(inputPath)
	if err != nil {
		return false
	}
	defer buf.Close()

	r := bufseekio.NewReadSeeker(buf, 128*1024, 4)
	path := mp4.BoxPath{
		mp4.BoxTypeMoov(),
		mp4.BoxTypeTrak(),
		mp4.BoxTypeMdia(),
		mp4.BoxTypeHdlr(),
	}
	boxes, err := mp4.ExtractBoxesWithPayload(r, nil, []mp4.BoxPath{path})
	if err != nil {
		return false
	}
	for _, bi := range boxes {
		hdlr, ok := bi.Payload.(*mp4.Hdlr)
		if !ok {
			continue
		}
		name := strings.TrimRight(hdlr.Name, "\x00") // null-terminated
		if strings.Contains(name, "Twitter-vork") {
			return true
		}
	}
	return false
}
