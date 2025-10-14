package util

import (
	"crypto/rand"
	"errors"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/govdbot/govd/internal/config"
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
		return "", errors.New("invalid domain structure")
	}
	return parts[0], nil
}

func ExceedsMaxFileSize(fileSize int32) bool {
	return fileSize > config.Env.MaxFileSize
}

func ExceedsMaxDuration(duration int32) bool {
	return duration > int32(config.Env.MaxDuration.Seconds())
}

func CleanupDownloads() {
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
		os.Remove(filePath)
	}
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
