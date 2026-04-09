package observability

import (
	"errors"
	"fmt"
	"net/http"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Metrics holds core Prometheus metrics used by go-core.
type Metrics struct {
	RequestTotal           *prometheus.CounterVec
	RequestDuration        *prometheus.HistogramVec
	HTTPRequestTotal       *prometheus.CounterVec
	HTTPRequestDuration    *prometheus.HistogramVec
	ServiceTotal           *prometheus.CounterVec
	ServiceDuration        *prometheus.HistogramVec
	DBTotal                *prometheus.CounterVec
	DBDuration             *prometheus.HistogramVec
	MessagePublishTotal    *prometheus.CounterVec
	MessageConsumeTotal    *prometheus.CounterVec
	MessageProcessDuration *prometheus.HistogramVec
	OutboxBatchTotal       *prometheus.CounterVec
	OutboxBatchDuration    *prometheus.HistogramVec
	OutboxBatchSize        *prometheus.HistogramVec
	TransactionTotal       *prometheus.CounterVec
}

var (
	metricsOnce sync.Once
	metricsInst *Metrics
)

// NewMetrics initializes and registers core metrics.
func NewMetrics() *Metrics {
	metricsOnce.Do(func() {
		metricsInst = &Metrics{
			RequestTotal: registerOrReuseCounterVec(
				prometheus.CounterOpts{
					Name: "app_request_total",
					Help: "Total number of incoming requests.",
				},
				[]string{"service", "method", "status"},
			),
			RequestDuration: registerOrReuseHistogramVec(
				prometheus.HistogramOpts{
					Name:    "app_request_duration_seconds",
					Help:    "Request duration in seconds.",
					Buckets: prometheus.DefBuckets,
				},
				[]string{"service", "method"},
			),
			HTTPRequestTotal: registerOrReuseCounterVec(
				prometheus.CounterOpts{
					Name: "app_http_request_total",
					Help: "Total number of HTTP requests handled by gateway.",
				},
				[]string{"service", "method", "route", "status"},
			),
			HTTPRequestDuration: registerOrReuseHistogramVec(
				prometheus.HistogramOpts{
					Name:    "app_http_request_duration_seconds",
					Help:    "HTTP request duration in seconds handled by gateway.",
					Buckets: prometheus.DefBuckets,
				},
				[]string{"service", "method", "route"},
			),
			ServiceTotal: registerOrReuseCounterVec(
				prometheus.CounterOpts{
					Name: "app_service_operation_total",
					Help: "Total number of structured service operations.",
				},
				[]string{"service", "operation", "status"},
			),
			ServiceDuration: registerOrReuseHistogramVec(
				prometheus.HistogramOpts{
					Name:    "app_service_operation_duration_seconds",
					Help:    "Structured service operation duration in seconds.",
					Buckets: prometheus.DefBuckets,
				},
				[]string{"service", "operation"},
			),
			DBTotal: registerOrReuseCounterVec(
				prometheus.CounterOpts{
					Name: "app_db_operation_total",
					Help: "Total number of structured database operations.",
				},
				[]string{"service", "db_name", "operation", "status"},
			),
			DBDuration: registerOrReuseHistogramVec(
				prometheus.HistogramOpts{
					Name:    "app_db_operation_duration_seconds",
					Help:    "Structured database operation duration in seconds.",
					Buckets: prometheus.DefBuckets,
				},
				[]string{"service", "db_name", "operation"},
			),
			MessagePublishTotal: registerOrReuseCounterVec(
				prometheus.CounterOpts{
					Name: "app_message_publish_total",
					Help: "Total number of message publish attempts by final status.",
				},
				[]string{"service", "topic", "status"},
			),
			MessageConsumeTotal: registerOrReuseCounterVec(
				prometheus.CounterOpts{
					Name: "app_message_consume_total",
					Help: "Total number of message consume attempts by final status.",
				},
				[]string{"service", "topic", "group", "status"},
			),
			MessageProcessDuration: registerOrReuseHistogramVec(
				prometheus.HistogramOpts{
					Name:    "app_message_process_duration_seconds",
					Help:    "Message handler processing duration in seconds.",
					Buckets: prometheus.DefBuckets,
				},
				[]string{"service", "topic", "group"},
			),
			OutboxBatchTotal: registerOrReuseCounterVec(
				prometheus.CounterOpts{
					Name: "app_outbox_batch_total",
					Help: "Total number of outbox batch executions by status.",
				},
				[]string{"service", "status"},
			),
			OutboxBatchDuration: registerOrReuseHistogramVec(
				prometheus.HistogramOpts{
					Name:    "app_outbox_batch_duration_seconds",
					Help:    "Outbox batch execution duration in seconds.",
					Buckets: prometheus.DefBuckets,
				},
				[]string{"service"},
			),
			OutboxBatchSize: registerOrReuseHistogramVec(
				prometheus.HistogramOpts{
					Name:    "app_outbox_batch_size",
					Help:    "Outbox batch size distribution.",
					Buckets: []float64{1, 5, 10, 25, 50, 100, 250, 500},
				},
				[]string{"service"},
			),
			TransactionTotal: registerOrReuseCounterVec(
				prometheus.CounterOpts{
					Name: "app_transaction_total",
					Help: "Total number of business transactions.",
				},
				[]string{"service", "operation", "status"},
			),
		}
	})

	return metricsInst
}

// Handler returns HTTP handler for Prometheus scraping.
func Handler() http.Handler {
	return promhttp.Handler()
}

func registerOrReuseCounterVec(
	opts prometheus.CounterOpts,
	labels []string,
) *prometheus.CounterVec {
	cv := prometheus.NewCounterVec(opts, labels)
	if err := prometheus.Register(cv); err != nil {
		var already prometheus.AlreadyRegisteredError
		if errors.As(err, &already) {
			if existing, ok := already.ExistingCollector.(*prometheus.CounterVec); ok {
				return existing
			}
		}
		panic(fmt.Sprintf("observability: register counter %q failed: %v", opts.Name, err))
	}
	return cv
}

func registerOrReuseHistogramVec(
	opts prometheus.HistogramOpts,
	labels []string,
) *prometheus.HistogramVec {
	hv := prometheus.NewHistogramVec(opts, labels)
	if err := prometheus.Register(hv); err != nil {
		var already prometheus.AlreadyRegisteredError
		if errors.As(err, &already) {
			if existing, ok := already.ExistingCollector.(*prometheus.HistogramVec); ok {
				return existing
			}
		}
		panic(fmt.Sprintf("observability: register histogram %q failed: %v", opts.Name, err))
	}
	return hv
}
