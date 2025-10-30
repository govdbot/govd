package bot

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	totalRequests = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "gotgbot",
			Name:      "requests_total",
			Help:      "Number of requests made to the bot API.",
		},
		[]string{
			"api_method",
		},
	)
	totalHTTPErrors = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "gotgbot",
			Name:      "http_request_errors_total",
			Help:      "Number of HTTP errors obtained.",
		},
		[]string{
			"api_method",
		},
	)
	totalAPIErrors = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "gotgbot",
			Name:      "api_request_errors_total",
			Help:      "Number of bot API errors obtained.",
		},
		[]string{
			"api_method",
			"api_status_code",
			"description",
		},
	)
	requestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "gotgbot",
			Name:      "api_request_time_seconds",
			Help:      "Duration of requests made to the bot API.",
		},
		[]string{
			"api_method",
		},
	)
)
