package util

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/govdbot/govd/internal/config"
	"github.com/govdbot/govd/internal/database"
	"github.com/govdbot/govd/internal/logger"
	"golang.org/x/net/publicsuffix"
)

func GetNamedGroups(re *regexp.Regexp, str string) map[string]string {
	match := re.FindStringSubmatch(str)
	names := re.SubexpNames()
	result := make(map[string]string)
	for i, name := range names {
		if i < len(match) && name != "" {
			result[name] = match[i]
		}
		result["match"] = match[0]
	}
	return result
}

func ExtractBaseHost(rawURL string) (string, error) {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}
	host := parsedURL.Hostname()
	etld, err := publicsuffix.EffectiveTLDPlusOne(host)
	if err != nil {
		return "", err
	}
	parts := strings.Split(etld, ".")
	if len(parts) == 0 {
		return "", fmt.Errorf("invalid domain structure")
	}
	return parts[0], nil
}

func ExceedsMaxFileSize(fileSize int64) bool {
	return fileSize > config.Env.MaxFileSize
}

func ExceedsMaxDuration(duration int32) bool {
	return duration > int32(config.Env.MaxDuration.Seconds())
}

func CleanupDownloads(ignoreTime bool) {
	logger.L.Debug("cleaning up downloads directory")

	if config.Env == nil || config.Env.DownloadsDirectory == "" {
		return
	}
	path := config.Env.DownloadsDirectory
	files, err := os.ReadDir(path)
	if err != nil {
		return
	}
	for _, file := range files {
		filePath := filepath.Join(path, file.Name())
		if err != nil {
			continue
		}
		info, err := file.Info()
		if err != nil {
			continue
		}
		modTime := info.ModTime()
		if ignoreTime || time.Since(modTime) > 10*time.Minute {
			if file.IsDir() {
				os.RemoveAll(filePath)
			} else {
				os.Remove(filePath)
			}
		}
	}
}

func CleanupDownloadsJob() {
	CleanupDownloads(true) // initial cleanup on startup

	tikcer := time.NewTicker(10 * time.Minute)
	go func() {
		for range tikcer.C {
			CleanupDownloads(false)
		}
	}()
}

func CheckFFmpeg() bool {
	_, err := exec.LookPath("ffmpeg")
	return err == nil
}

func RandomBase64(length int) string {
	const letters = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-_"
	const mask = 63 // 6 bits, since len(letters) == 64

	result := make([]byte, length)
	random := make([]byte, length)
	_, err := rand.Read(random)
	if err != nil {
		return strings.Repeat("A", length)
	}
	for i, b := range random {
		result[i] = letters[int(b)&mask]
	}
	return string(result)
}

func RandomAlphaString(length int) string {
	const letters = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
	const lettersLen = byte(len(letters))
	const maxByte = 255 - (255 % lettersLen) // 255 - (255 % 52) = 255 - 47 = 208

	result := make([]byte, length)
	i := 0
	for i < length {
		b := make([]byte, 1)
		_, err := rand.Read(b)
		if err != nil {
			return strings.Repeat("a", length)
		}
		if b[0] > maxByte {
			continue // avoid bias
		}
		result[i] = letters[b[0]%lettersLen]
		i++
	}
	return string(result)
}

func ParseHex(str string) ([]byte, error) {
	if strings.HasPrefix(str, "0x") || strings.HasPrefix(str, "0X") {
		str = str[2:]
	}
	iv, err := hex.DecodeString(str)
	if err != nil {
		return nil, fmt.Errorf("invalid hex IV: %w", err)
	}
	if len(iv) != 16 {
		return nil, fmt.Errorf("IV must be 16 bytes, got %d", len(iv))
	}

	return iv, nil
}

func ParseVideoCodec(codecs string) database.MediaCodec {
	codecs = strings.ToLower(codecs)
	switch {
	case strings.Contains(codecs, "avc") || strings.Contains(codecs, "h264"):
		return database.MediaCodecAvc
	case strings.Contains(codecs, "hvc") || strings.Contains(codecs, "h265") || strings.Contains(codecs, "hev1"):
		return database.MediaCodecHevc
	case strings.Contains(codecs, "av01"):
		return database.MediaCodecAv1
	case strings.Contains(codecs, "vp9"):
		return database.MediaCodecVp9
	case strings.Contains(codecs, "vp8"):
		return database.MediaCodecVp9
	default:
		return ""
	}
}

func ParseAudioCodec(codecs string) database.MediaCodec {
	codecs = strings.ToLower(codecs)
	switch {
	case strings.Contains(codecs, "mp4a"):
		return database.MediaCodecAac
	case strings.Contains(codecs, "opus"):
		return database.MediaCodecOpus
	case strings.Contains(codecs, "mp3"):
		return database.MediaCodecMp3
	case strings.Contains(codecs, "flac"):
		return database.MediaCodecFlac
	case strings.Contains(codecs, "vorbis"):
		return database.MediaCodecVorbis
	default:
		return ""
	}
}

func UnescaepURL(url string) string {
	return strings.ReplaceAll(url, "&amp;", "&")
}
