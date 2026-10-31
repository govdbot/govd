package util

import (
	"bytes"
	"encoding/binary"
	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/govdbot/govd/internal/logger"
	"go.uber.org/zap"
	"hash/crc32"
	"image"
	"image/jpeg"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestURLMessageRepresentations(t *testing.T) {
	raw := "https://www.instagram.com/p/B/"
	msg := &gotgbot.Message{Text: "😀 " + raw, Entities: []gotgbot.MessageEntity{{Type: "url", Offset: 3, Length: int64(len(raw))}}}
	if URLFromMessage(msg) != raw {
		t.Fatal("UTF16 URL failed")
	}
	msg = &gotgbot.Message{Caption: raw, CaptionEntities: []gotgbot.MessageEntity{{Type: "url", Length: int64(len(raw))}}}
	if URLFromMessage(msg) != raw {
		t.Fatal("caption URL failed")
	}
	msg = &gotgbot.Message{Text: "link", Entities: []gotgbot.MessageEntity{{Type: "text_link", Length: 4, Url: raw}}}
	if URLFromMessage(msg) != raw {
		t.Fatal("hidden link failed")
	}
	msg.Entities = append(msg.Entities, msg.Entities[0])
	if len(URLsFromMessage(msg)) != 1 {
		t.Fatal("duplicate not removed")
	}
	for _, entity := range []gotgbot.MessageEntity{{Type: "url", Offset: -1, Length: 4}, {Type: "url", Offset: 1 << 62, Length: 8}, {Type: "url", Offset: 1, Length: 1 << 62}} {
		if URLFromMessage(&gotgbot.Message{Text: "abc", Entities: []gotgbot.MessageEntity{entity}}) != "" {
			t.Fatal("malformed entity accepted")
		}
	}
}
func TestImageDimensionsRejectedBeforePixelDecode(t *testing.T) {
	logger.L = zap.NewNop().Sugar()
	b := []byte{137, 80, 78, 71, 13, 10, 26, 10}
	data := make([]byte, 13)
	binary.BigEndian.PutUint32(data[:4], 30000)
	binary.BigEndian.PutUint32(data[4:8], 30000)
	data[8] = 8
	data[9] = 6
	chunk := append([]byte("IHDR"), data...)
	length := make([]byte, 4)
	binary.BigEndian.PutUint32(length, 13)
	b = append(b, length...)
	b = append(b, chunk...)
	binary.BigEndian.PutUint32(length, crc32.ChecksumIEEE(chunk))
	b = append(b, length...)
	b = append(b, make([]byte, 128)...)
	if _, _, err := DecodeImage(bytes.NewReader(b), 0); err == nil || !strings.Contains(err.Error(), "dimensions") {
		t.Fatalf("unexpected result %v", err)
	}
}
func TestResizedJPEGActuallyWritesSmallImage(t *testing.T) {
	logger.L = zap.NewNop().Sugar()
	var b bytes.Buffer
	jpeg.Encode(&b, image.NewRGBA(image.Rect(0, 0, 640, 640)), nil)
	path := filepath.Join(t.TempDir(), "thumb.jpg")
	if _, err := ImgToJPEG(bytes.NewReader(b.Bytes()), path, 320); err != nil {
		t.Fatal(err)
	}
	file, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	cfg, err := jpeg.DecodeConfig(file)
	if err != nil || cfg.Width != 320 || cfg.Height != 320 {
		t.Fatalf("not resized: %+v %v", cfg, err)
	}
}
