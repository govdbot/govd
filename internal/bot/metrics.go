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
			Namespace: metricsNamespace,
			Name:      "requests_total",
			Help:      "Number of requests made to the bot API.",
		},
		[]string{
			apiMethodField,
		},
	)
	totalHTTPErrors = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Name:      "http_request_errors_total",
			Help:      "Number of HTTP errors obtained.",
		},
		[]string{
			apiMethodField,
		},
	)
	totalAPIErrors = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Name:      "api_request_errors_total",
			Help:      "Number of bot API errors obtained.",
		},
		[]string{
			apiMethodField,
			"api_status_code",
			"description",
		},
	)
	requestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: metricsNamespace,
			Name:      "api_request_time_seconds",
			Help:      "Duration of requests made to the bot API.",
		},
		[]string{
			apiMethodField,
		},
	)
	totalUpdates = promauto.NewCounter(
		prometheus.CounterOpts{
			Namespace: metricsNamespace,
			Name:      "updates_total",
			Help:      "Number of incoming updates.",
		},
	)
	updateProcessingDuration = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Namespace: metricsNamespace,
			Name:      "update_processing_time_seconds",
			Help:      "Time to process each update.",
		},
	)
	bufferedUpdates = promauto.NewGauge(
		prometheus.GaugeOpts{
			Namespace: metricsNamespace,
			Name:      "buffered_updates",
			Help:      "Number of updates currently buffered in the dispatcher limiter channel.",
		},
	)
	bufferedUpdatesLimit = promauto.NewGauge(
		prometheus.GaugeOpts{
			Namespace: metricsNamespace,
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
