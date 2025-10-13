package soundcloud

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"github.com/govdbot/govd/internal/logger"
	"github.com/govdbot/govd/internal/models"
)

var clientIDPattern = regexp.MustCompile(`"clientId"\s*:\s*"([0-9a-zA-Z]{32})"`)

func ResolveURL(targetURL string) string {
	return apiHostname +
		"resolve?url=" +
		url.PathEscape(targetURL)
}

func GetThumbnailURL(thumbnailURL string) string {
	if thumbnailURL == "" {
		return ""
	}
	return strings.Replace(
		thumbnailURL,
		"-large", "-original", 1,
	)
}

func GetClientID(ctx *models.ExtractorContext) (string, error) {
	resp, err := ctx.Fetch(
		http.MethodGet,
		baseURL, nil,
	)
	if err != nil {
		return "", fmt.Errorf("failed to get main page: %w", err)
	}
	defer resp.Body.Close()

	logger.WriteFile("soundcloud_main_page", resp)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	clientMatch := clientIDPattern.FindSubmatch(body)
	if clientMatch != nil {
		clientID := string(clientMatch[1])
		return clientID, nil
	}

	return "", fmt.Errorf("client ID not found")
}
