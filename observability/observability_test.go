package observability

import (
	"context"
	"fmt"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/yogayulanda/go-core/config"
)

func TestWithAndGetRequestID(t *testing.T) {
	ctx := WithRequestID(context.Background(), "req-123")
	if got := GetRequestID(ctx); got != "req-123" {
		t.Fatalf("expected req-123, got %s", got)
	}
}

func TestGenerateRequestID(t *testing.T) {
	a := GenerateRequestID()
	b := GenerateRequestID()

	if a == "" || b == "" {
		t.Fatalf("request id must not be empty")
	}
	if a == b {
		t.Fatalf("request id must be unique")
	}
}

func TestNewMetrics_Singleton(t *testing.T) {
	a := NewMetrics()
	b := NewMetrics()

	if a == nil || b == nil {
		t.Fatalf("metrics must not be nil")
	}
	if a != b {
		t.Fatalf("metrics must be singleton")
	}
}

func TestMetricsHandler(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/metrics", nil)
	Handler().ServeHTTP(rec, req)

	if rec.Code != 200 {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestInitTracing_Disabled_NoError(t *testing.T) {
	cfg := &config.Config{
		App: config.AppConfig{
			ServiceName: "obs-test",
			Environment: "test",
		},
	}

	shutdown, err := InitTracing(context.Background(), cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if shutdown == nil {
		t.Fatalf("expected shutdown function")
	}
	if err := shutdown(context.Background()); err != nil {
		t.Fatalf("unexpected shutdown error: %v", err)
	}
}

func TestInitTracing_NilConfig_ReturnError(t *testing.T) {
	_, err := InitTracing(context.Background(), nil)
	if err == nil {
		t.Fatalf("expected error for nil config")
	}
}

func TestBuildOTLPTLSConfig_InvalidPath_ReturnError(t *testing.T) {
	_, err := buildOTLPTLSConfig("/path/not/found.pem")
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestBuildOTLPTLSConfig_InvalidPEM_ReturnError(t *testing.T) {
	file, err := os.CreateTemp("", "otlp-ca-*.pem")
	if err != nil {
		t.Fatalf("create temp file failed: %v", err)
	}
	defer os.Remove(file.Name())
	if _, err := file.WriteString("not-a-pem"); err != nil {
		t.Fatalf("write temp file failed: %v", err)
	}
	_ = file.Close()

	_, err = buildOTLPTLSConfig(file.Name())
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestRegisterOrReuseCounterVec_AlreadyRegistered_SameType(t *testing.T) {
	name := fmt.Sprintf("go_core_test_counter_%d", time.Now().UnixNano())

	first := registerOrReuseCounterVec(
		prometheus.CounterOpts{Name: name, Help: "test"},
		[]string{"label"},
	)
	second := registerOrReuseCounterVec(
		prometheus.CounterOpts{Name: name, Help: "test"},
		[]string{"label"},
	)

	if first == nil || second == nil {
		t.Fatalf("counter vec must not be nil")
	}
	if first != second {
		t.Fatalf("expected existing collector to be reused")
	}
}

func TestRegisterOrReuseCounterVec_AlreadyRegistered_DifferentType_Panic(t *testing.T) {
	name := fmt.Sprintf("go_core_test_counter_conflict_%d", time.Now().UnixNano())
	gauge := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: name,
		Help: "conflict",
	}, []string{"label"})
	if err := prometheus.Register(gauge); err != nil {
		t.Fatalf("register gauge failed: %v", err)
	}

	defer func() {
		if recover() == nil {
			t.Fatalf("expected panic for collector type mismatch")
		}
	}()
	registerOrReuseCounterVec(
		prometheus.CounterOpts{Name: name, Help: "conflict"},
		[]string{"label"},
	)
}

func TestRegisterOrReuseHistogramVec_AlreadyRegistered_DifferentType_Panic(t *testing.T) {
	name := fmt.Sprintf("go_core_test_histogram_conflict_%d", time.Now().UnixNano())
	summary := prometheus.NewSummaryVec(prometheus.SummaryOpts{
		Name: name,
		Help: "conflict",
	}, []string{"label"})
	if err := prometheus.Register(summary); err != nil {
		t.Fatalf("register summary failed: %v", err)
	}

	defer func() {
		if recover() == nil {
			t.Fatalf("expected panic for collector type mismatch")
		}
	}()
	registerOrReuseHistogramVec(
		prometheus.HistogramOpts{Name: name, Help: "conflict"},
		[]string{"label"},
	)
}

func TestRegisterOrReuseCounterVec_AlreadyRegistered_DifferentType_Isolated(t *testing.T) {
	// Verifies the panic path is isolated and does not affect normal unique registration.
	name := fmt.Sprintf("go_core_test_counter_unique_%d", time.Now().UnixNano())
	counter := registerOrReuseCounterVec(
		prometheus.CounterOpts{Name: name, Help: "unique"},
		[]string{"label"},
	)
	if counter == nil {
		t.Fatalf("expected counter vec")
	}
}
