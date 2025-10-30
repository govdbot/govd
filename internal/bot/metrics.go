package bot

import (
	"time"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
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
	totalUpdates = promauto.NewCounter(
		prometheus.CounterOpts{
			Namespace: "gotgbot",
			Name:      "updates_total",
			Help:      "Number of incoming updates.",
		},
	)
	updateProcessingDuration = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Namespace: "gotgbot",
			Name:      "update_processing_time_seconds",
			Help:      "Time to process each update.",
		},
	)
	bufferedUpdates = promauto.NewGauge(
		prometheus.GaugeOpts{
			Namespace: "gotgbot",
			Name:      "buffered_updates",
			Help:      "Number of updates currently buffered in the dispatcher limiter channel.",
		},
	)
	bufferedUpdatesLimit = promauto.NewGauge(
		prometheus.GaugeOpts{
			Namespace: "gotgbot",
			Name:      "buffered_updates_limit",
			Help:      "Maximum number of buffered updates in the limiter channel.",
		},
	)
)

var _ ext.Processor = metricsProcessor{}

type metricsProcessor struct {
	processor ext.Processor
}

func (m metricsProcessor) ProcessUpdate(d *ext.Dispatcher, b *gotgbot.Bot, ctx *ext.Context) error {
	totalUpdates.Inc()
	timer := prometheus.NewTimer(updateProcessingDuration)
	defer timer.ObserveDuration()

	return m.processor.ProcessUpdate(d, b, ctx)
}

func monitorDispatcherBuffer(d *ext.Dispatcher) {
	bufferedUpdatesLimit.Set(float64(d.MaxUsage()))

	for {
		bufferedUpdates.Set(float64(d.CurrentUsage()))
		time.Sleep(time.Second)
	}
}
