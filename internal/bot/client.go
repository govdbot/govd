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
	var timer *prometheus.Timer
	totalRequests.WithLabelValues(method).Inc()
	timer = prometheus.NewTimer(requestDuration.With(prometheus.Labels{
		"api_method": method,
	}))

	if strings.HasPrefix(method, "send") || method == "copyMessage" {
		params["allow_sending_without_reply"] = "true"
	}
	if strings.HasPrefix(method, "send") || strings.HasPrefix(method, "edit") {
		params["parse_mode"] = gotgbot.ParseModeHTML
	}

	val, err := b.BotClient.RequestWithContext(ctx, token, method, params, opts)
	timer.ObserveDuration()
	if err != nil {
		tgErr := &gotgbot.TelegramError{}
		if errors.As(err, &tgErr) {
			totalAPIErrors.WithLabelValues(
				method, strconv.Itoa(tgErr.Code),
				tgErr.Description,
			).Inc()
		} else {
			totalHTTPErrors.WithLabelValues(method).Inc()
		}
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
