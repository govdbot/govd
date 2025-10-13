package util

import (
	"errors"
	"net/url"
	"os"
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
