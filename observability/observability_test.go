package observability

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"
	"unicode"
	"unicode/utf8"

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

func TestRequestIDValidation(t *testing.T) {
	for _, requestID := range []string{"a", "abc-123.X_y", "A" + strings.Repeat("b", 63)} {
		if !IsValidRequestID(requestID) {
			t.Errorf("expected request ID %q to be valid", requestID)
		}
	}
	for _, requestID := range []string{"", "../etc", strings.Repeat("a", 65), "line\nbreak", "ümlaut"} {
		if IsValidRequestID(requestID) {
			t.Errorf("expected request ID %q to be invalid", requestID)
		}
	}
}

func TestIsValidRequestIDMatchesPreviousPredicate(t *testing.T) {
	for value := 0; value <= 255; value++ {
		candidate := string([]byte{byte(value)})
		if got, want := IsValidRequestID(candidate), previousIsValidRequestID(candidate); got != want {
			t.Errorf("byte 0x%02x: IsValidRequestID=%t, previous predicate=%t", value, got, want)
		}
	}

	for _, value := range []rune{'é', '中', 'Ω', '\u0301', '😀', utf8.RuneError} {
		candidate := string(value)
		if got, want := IsValidRequestID(candidate), previousIsValidRequestID(candidate); got != want {
			t.Errorf("rune %U: IsValidRequestID=%t, previous predicate=%t", value, got, want)
		}
	}
}

func previousIsValidRequestID(requestID string) bool {
	if len(requestID) == 0 || len(requestID) > 64 {
		return false
	}
	for _, r := range requestID {
		// Keep the historical expression intact as the equivalence-test oracle.
		//nolint:staticcheck // Preserve the exact pre-change predicate.
		if !(unicode.IsLetter(r) && r <= unicode.MaxASCII) && !(r >= '0' && r <= '9') && r != '.' && r != '_' && r != '-' {
			return false
		}
	}
	return true
}

func TestFieldBagSetAndRestoreIncomingIdentity(t *testing.T) {
	ctx := EnsureFieldBag(context.Background())
	if err := RegisterPropagatedLabel("client_app"); err != nil {
		t.Fatalf("register propagated label: %v", err)
	}
	RestoreIncomingIdentity(ctx, "user-1", map[string]string{"client_app": "app-a"})
	if got := GetUserID(ctx); got != "user-1" {
		t.Fatalf("unexpected user id: %q", got)
	}
	if got := Labels(ctx)["client_app"]; got != "app-a" {
		t.Fatalf("unexpected propagated label: %q", got)
	}
	if err := SetLabel(ctx, "password", "secret"); err != nil {
		t.Fatalf("set label: %v", err)
	}
	labels := Labels(ctx)
	labels["client_app"] = "mutated-copy"
	if got := Labels(ctx)["client_app"]; got != "app-a" {
		t.Fatalf("Labels must return a copy; got %q", got)
	}
}

func TestFieldBagConcurrentUpdates(t *testing.T) {
	ctx := EnsureFieldBag(context.Background())
	var workers sync.WaitGroup
	for i := range 16 {
		workers.Add(1)
		go func(index int) {
			defer workers.Done()
			_ = SetUserID(ctx, fmt.Sprintf("user-%d", index))
			_ = SetLabel(ctx, fmt.Sprintf("label%d", index), "value")
			_ = Labels(ctx)
		}(i)
	}
	workers.Wait()
	if len(Labels(ctx)) != 16 {
		t.Fatalf("expected all concurrent labels, got %d", len(Labels(ctx)))
	}
}

func TestNewMetrics_Singleton(t *testing.T) {
	a := NewMetrics(true)
	b := NewMetrics(true)

	if a == nil || b == nil {
		t.Fatalf("metrics must not be nil")
	}
	if a == b || a.RequestTotal != b.RequestTotal {
		t.Fatalf("metrics views should share collectors but retain per-runtime enable state")
	}
	if NewMetrics().Enabled || NewMetrics(false).Enabled {
		t.Fatal("metrics enable state was not preserved by views")
	}
	if a.ServiceTotal == nil || a.ServiceDuration == nil {
		t.Fatalf("service metrics must not be nil")
	}
	if a.DBTotal == nil || a.DBDuration == nil {
		t.Fatalf("db metrics must not be nil")
	}
	if a.MessagePublishTotal == nil || a.MessageConsumeTotal == nil || a.MessageProcessDuration == nil {
		t.Fatalf("messaging metrics must not be nil")
	}
	if a.OutboxBatchTotal == nil || a.OutboxBatchDuration == nil || a.OutboxBatchSize == nil {
		t.Fatalf("outbox metrics must not be nil")
	}
}

func TestMetricsHandler(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/metrics", nil)
	Handler(true).ServeHTTP(rec, req)

	if rec.Code != 200 {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestMetricsHandlerDisabledByDefault(t *testing.T) {
	recorder := httptest.NewRecorder()
	Handler().ServeHTTP(recorder, httptest.NewRequest("GET", "/metrics", nil))
	if recorder.Code != http.StatusNotFound {
		t.Fatalf("disabled metrics handler returned %d, want 404", recorder.Code)
	}
}

func TestMetricsRegistryDoesNotExposeTransactionCounter(t *testing.T) {
	metrics := NewMetrics(true)
	if !metrics.Enabled {
		t.Fatal("expected explicitly enabled metrics view")
	}
	families, err := prometheus.DefaultGatherer.Gather()
	if err != nil {
		t.Fatal(err)
	}
	for _, family := range families {
		if family.GetName() == "app_transaction_total" {
			t.Fatal("app_transaction_total must be removed")
		}
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
