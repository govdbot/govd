package util

import (
	"errors"
	"net/url"
	"regexp"
	"strings"

	"github.com/govdbot/govd/internal/config"
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

func ExceedsMaxFileSize(fileSize int64) bool {
	return fileSize > config.Env.MaxFileSize
}

func ExceedsMaxDuration(duration int32) bool {
	return duration > int32(config.Env.MaxDuration.Seconds())
}
