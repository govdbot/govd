package util

import (
	"errors"
	"net/url"
	"regexp"
	"strings"

	"golang.org/x/net/publicsuffix"
)

func ChunkedSlice[T any](slice []T, size int) [][]T {
	chunks := make([][]T, 0, (len(slice)+size-1)/size)
	for i := 0; i < len(slice); i += size {
		end := min(i+size, len(slice))
		chunks = append(chunks, slice[i:end])
	}
	return chunks
}

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
