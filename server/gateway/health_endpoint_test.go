package gateway

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"

	"github.com/yogayulanda/go-core/app"
	"github.com/yogayulanda/go-core/config"
	"github.com/yogayulanda/go-core/version"
)

func TestReadyEndpoint_ReadyJSON(t *testing.T) {
	application := newTestApp(t, &config.Config{
		App: config.AppConfig{
			ServiceName:     "gateway-ready-test",
			Environment:     "test",
			LogLevel:        "error",
			ShutdownTimeout: time.Second,
		},
		Databases: map[string]config.DBConfig{},
	})

	mux := runtime.NewServeMux()
	if err := registerHealthEndpoints(mux, application); err != nil {
		t.Fatalf("register health endpoints failed: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/ready", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if got := rec.Header().Get("Content-Type"); got != "application/json" {
		t.Fatalf("unexpected content type: %s", got)
	}

	var body readinessReport
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal response failed: %v", err)
	}
	if body.Status != readinessStatusReady {
		t.Fatalf("expected %q, got %q", readinessStatusReady, body.Status)
	}

	redisCheck, ok := body.Checks["redis"]
	if !ok {
		t.Fatalf("expected redis check")
	}
	if redisCheck.Status != checkStatusSkipped {
		t.Fatalf("expected redis skipped, got %+v", redisCheck)
	}
}

func TestReadyEndpoint_NotReadyJSON(t *testing.T) {
	application := newTestApp(t, &config.Config{
		App: config.AppConfig{
			ServiceName:     "gateway-not-ready-test",
			Environment:     "test",
			LogLevel:        "error",
			ShutdownTimeout: time.Second,
		},
		Databases: map[string]config.DBConfig{},
		Kafka: config.KafkaConfig{
			Enabled: true,
			Brokers: []string{},
		},
	})

	mux := runtime.NewServeMux()
	if err := registerHealthEndpoints(mux, application); err != nil {
		t.Fatalf("register health endpoints failed: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/ready", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", rec.Code)
	}

	var body readinessReport
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal response failed: %v", err)
	}
	if body.Status != readinessStatusNotReady {
		t.Fatalf("expected %q, got %q", readinessStatusNotReady, body.Status)
	}

	kafkaCheck, ok := body.Checks["kafka"]
	if !ok {
		t.Fatalf("expected kafka check")
	}
	if kafkaCheck.Status != checkStatusDown || !kafkaCheck.Required {
		t.Fatalf("unexpected kafka check: %+v", kafkaCheck)
	}
}

func TestHealthEndpoint_OK(t *testing.T) {
	application := newTestApp(t, &config.Config{
		App: config.AppConfig{
			ServiceName:     "gateway-health-test",
			Environment:     "test",
			LogLevel:        "error",
			ShutdownTimeout: time.Second,
		},
		Databases: map[string]config.DBConfig{},
	})

	mux := runtime.NewServeMux()
	if err := registerHealthEndpoints(mux, application); err != nil {
		t.Fatalf("register health endpoints failed: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if strings.TrimSpace(rec.Body.String()) != "ok" {
		t.Fatalf("unexpected body: %q", rec.Body.String())
	}
}

func TestVersionEndpoint_JSON(t *testing.T) {
	origVersion := version.Version
	origCommit := version.Commit
	origBuildDate := version.BuildDate
	defer func() {
		version.Version = origVersion
		version.Commit = origCommit
		version.BuildDate = origBuildDate
	}()

	version.Version = "1.0.0"
	version.Commit = "deadbeef"
	version.BuildDate = "2026-04-10T00:00:00Z"

	mux := runtime.NewServeMux()
	if err := registerVersionEndpoint(mux); err != nil {
		t.Fatalf("register version endpoint failed: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/version", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if got := rec.Header().Get("Content-Type"); got != "application/json" {
		t.Fatalf("unexpected content type: %s", got)
	}

	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("invalid json response: %v", err)
	}
	if got := payload["version"]; got != version.Version {
		t.Fatalf("unexpected version: %v", got)
	}
	if got := payload["commit"]; got != version.Commit {
		t.Fatalf("unexpected commit: %v", got)
	}
	if got := payload["build_date"]; got != version.BuildDate {
		t.Fatalf("unexpected build_date: %v", got)
	}
}

func TestMetricsEndpoint_OK(t *testing.T) {
	mux := runtime.NewServeMux()
	if err := registerMetricsEndpoint(mux); err != nil {
		t.Fatalf("register metrics endpoint failed: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if got := rec.Header().Get("Content-Type"); !strings.HasPrefix(got, "text/plain") {
		t.Fatalf("unexpected content type: %s", got)
	}
	if len(rec.Body.Bytes()) == 0 {
		t.Fatalf("expected metrics payload")
	}
}

func newTestApp(t *testing.T, cfg *config.Config) *app.App {
	t.Helper()

	application, err := app.New(context.Background(), cfg)
	if err != nil {
		t.Fatalf("init app failed: %v", err)
	}

	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_ = application.Lifecycle().Shutdown(ctx)
	})

	return application
}
