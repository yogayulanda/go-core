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
	RequestTotal        *prometheus.CounterVec
	RequestDuration     *prometheus.HistogramVec
	HTTPRequestTotal    *prometheus.CounterVec
	HTTPRequestDuration *prometheus.HistogramVec
	TransactionTotal    *prometheus.CounterVec
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
