package util

import (
	"crypto/rand"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/govdbot/govd/config"
	"github.com/govdbot/govd/models"
	"github.com/govdbot/govd/util/networking"

	"github.com/pkg/errors"
	"go.uber.org/zap"
	"golang.org/x/net/publicsuffix"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/aki237/nscjar"
)

var cookiesCache = make(map[string][]*http.Cookie)

func ExceedsMaxFileSize(fileSize int64) bool {
	return fileSize > config.Env.MaxFileSize
}

func ExceedsMaxDuration(duration int64) bool {
	return duration > int64(config.Env.MaxDuration.Seconds())
}

func FetchPage(
	client models.HTTPClient,
	method string,
	url string,
	body io.Reader,
	headers map[string]string,
	cookies []*http.Cookie,
) (*http.Response, error) {
	if client == nil {
		client = networking.GetDefaultHTTPClient()
	}
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	if req.Header.Get("User-Agent") == "" {
		req.Header.Set("User-Agent", ChromeUA)
	}
	for _, cookie := range cookies {
		req.AddCookie(cookie)
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	return resp, nil
}

func GetLocationURL(
	client models.HTTPClient,
	url string,
	headers map[string]string,
	cookies []*http.Cookie,
) (string, error) {
	if client == nil {
		client = networking.GetDefaultHTTPClient()
	}
	setupRequest := func(method, url string) (*http.Request, error) {
		req, err := http.NewRequest(method, url, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}
		for k, v := range headers {
			req.Header.Set(k, v)
		}
		if req.Header.Get("User-Agent") == "" {
			req.Header.Set("User-Agent", ChromeUA)
		}
		for _, cookie := range cookies {
			req.AddCookie(cookie)
		}
		return req, nil
	}

	// try HEAD first
	req, err := setupRequest(http.MethodHead, url)
	if err != nil {
		return "", err
	}
	resp, err := client.Do(req)
	if err == nil {
		resp.Body.Close()
		url := resp.Request.URL.String()
		zap.S().Debugf("head response url: %s", url)
		return url, nil
	}

	// fallback to GET
	req, err = setupRequest(http.MethodGet, url)
	if err != nil {
		return "", err
	}
	resp, err = client.Do(req)
	if err == nil {
		resp.Body.Close()
		url := resp.Request.URL.String()
		zap.S().Debugf("get response url: %s", url)
		return url, nil
	}
	return "", errors.New("failed to get location url")
}

func IsUserAdmin(
	bot *gotgbot.Bot,
	chatID int64,
	userID int64,
) bool {
	chatMember, err := bot.GetChatMember(chatID, userID, nil)
	if err != nil {
		return false
	}
	if chatMember == nil {
		return false
	}
	status := chatMember.GetStatus()
	switch status {
	case "creator":
		return true
	case "administrator":
		if chatMember.MergeChatMember().CanChangeInfo {
			return true
		}
		return false
	}
	return false
}

func EscapeCaption(str string) string {
	// we wont use html.EscapeString
	// cuz it will escape all the characters
	// and we only need to escape < and >
	chars := map[string]string{
		"<": "&lt;",
		">": "&gt;",
	}
	for k, v := range chars {
		str = strings.ReplaceAll(str, k, v)
	}
	return str
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

func GetExtractorCookies(extractor *models.Extractor) []*http.Cookie {
	if extractor == nil {
		return nil
	}
	cookieFile := extractor.CodeName + ".txt"
	cookies, err := ParseCookieFile(cookieFile)
	if err != nil {
		return nil
	}
	return cookies
}

func ParseCookieFile(fileName string) ([]*http.Cookie, error) {
	cachedCookies, ok := cookiesCache[fileName]
	if ok {
		return cachedCookies, nil
	}
	cookiePath := filepath.Join("cookies", fileName)
	cookieFile, err := os.Open(cookiePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open cookie file: %w", err)
	}
	defer cookieFile.Close()

	var parser nscjar.Parser
	cookies, err := parser.Unmarshal(cookieFile)
	if err != nil {
		return nil, fmt.Errorf("failed to parse cookie file: %w", err)
	}
	cookiesCache[fileName] = cookies
	return cookies, nil
}

func FixURL(url string) string {
	return strings.ReplaceAll(url, "&amp;", "&")
}

func CleanupDownloadsDir() {
	dir := config.Env.DownloadsDirectory
	zap.S().Debug("cleaning up downloads directory")
	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if path == dir {
			return nil
		}
		if time.Since(info.ModTime()) > 30*time.Minute {
			if info.IsDir() {
				os.RemoveAll(path)
			} else {
				os.Remove(path)
			}
		}
		return nil
	})
}

func ExtractBaseHost(rawURL string) (string, error) {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return "", fmt.Errorf("failed to parse URL: %w", err)
	}
	host := parsedURL.Hostname()
	etld, err := publicsuffix.EffectiveTLDPlusOne(host)
	if err != nil {
		return "", fmt.Errorf("failed to get eTLD+1: %w", err)
	}
	parts := strings.Split(etld, ".")
	if len(parts) == 0 {
		return "", errors.New("invalid domain structure")
	}
	return parts[0], nil
}

func RedactURLs(text string) string {
	urlRegex := regexp.MustCompile(`https?://[^\s"'<>]+`)
	return urlRegex.ReplaceAllStringFunc(text, func(match string) string {
		return html.EscapeString("<redacted url>")
	})
}

func StartDownloadsCleanup() {
	ticker := time.NewTicker(1 * time.Hour)
	go func() {
		for {
			CleanupDownloadsDir()
			<-ticker.C
		}
	}()
}
