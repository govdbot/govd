package bot

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/govdbot/govd/internal/config"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/PaulSonOfLars/gotgbot/v2"
)

type Client struct {
	gotgbot.BotClient
}

func (b Client) RequestWithContext(
	ctx context.Context,
	token string,
	method string,
	params map[string]any,
	opts *gotgbot.RequestOpts,
) (json.RawMessage, error) {
	const maxRetries = 3

	if strings.HasPrefix(method, "send") || strings.HasPrefix(method, "edit") {
		params["parse_mode"] = gotgbot.ParseModeHTML
	}

	var val json.RawMessage
	var err error

	for attempt := range maxRetries {
		totalRequests.WithLabelValues(method).Inc()
		timer := prometheus.NewTimer(requestDuration.With(prometheus.Labels{
			"api_method": method,
		}))

		val, err = b.BotClient.RequestWithContext(ctx, token, method, params, opts)
		timer.ObserveDuration()

		if err == nil {
			return val, nil
		}

		var tgErr *gotgbot.TelegramError
		if !errors.As(err, &tgErr) {
			totalHTTPErrors.WithLabelValues(method).Inc()
			break
		}

		totalAPIErrors.WithLabelValues(
			method,
			strconv.Itoa(tgErr.Code),
			tgErr.Description,
		).Inc()

		if tgErr.ResponseParams == nil || tgErr.ResponseParams.RetryAfter <= 0 {
			break
		}

		if attempt >= maxRetries {
			break
		}

		retryDuration := time.Duration(tgErr.ResponseParams.RetryAfter) * time.Second
		time.Sleep(retryDuration)
	}

	return val, err
}

func NewBotClient() Client {
	return Client{
		BotClient: &gotgbot.BaseBotClient{
			Client: http.Client{
				Transport: &http.Transport{
					// avoid using proxy for telegram
					Proxy: func(_ *http.Request) (*url.URL, error) {
						return nil, nil
					},
				},
			},
			UseTestEnvironment: false,
			DefaultRequestOpts: &gotgbot.RequestOpts{
				Timeout: 10 * time.Minute,
				APIURL:  config.Env.BotAPIURL,
			},
		},
	}
}
