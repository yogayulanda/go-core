package gateway

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/yogayulanda/go-core/app"
	"github.com/yogayulanda/go-core/config"
	"github.com/yogayulanda/go-core/observability"
)

func TestPanicRecoveryWritesEnvelopeAndStackTrace(t *testing.T) {
	oldStdout := os.Stdout
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("create stdout pipe: %v", err)
	}
	os.Stdout = writer
	t.Cleanup(func() {
		os.Stdout = oldStdout
		_ = reader.Close()
		_ = writer.Close()
	})
	application, err := app.New(context.Background(), &config.Config{
		App:       config.AppConfig{ServiceName: "gateway-panic-test", Environment: "dev", LogLevel: "debug", ShutdownTimeout: time.Second},
		Databases: map[string]config.DBConfig{},
	})
	if err != nil {
		t.Fatalf("initialize app: %v", err)
	}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/panic", nil)
	ctx := observability.EnsureFieldBag(req.Context())
	ctx = observability.WithRequestID(ctx, "panic-1")
	req = req.WithContext(ctx)
	withPanicRecovery(application, http.HandlerFunc(func(http.ResponseWriter, *http.Request) { panic("panic detail") })).ServeHTTP(rec, req)
	_ = writer.Close()
	os.Stdout = oldStdout
	raw, err := io.ReadAll(reader)
	_ = reader.Close()
	if err != nil {
		t.Fatalf("read log output: %v", err)
	}
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected HTTP 500, got %d", rec.Code)
	}
	var body gatewayEnvelope
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode panic envelope: %v", err)
	}
	if body.ResponseCode != "CORE-9000" || body.Status != "ERROR" || body.RequestID != "panic-1" {
		t.Fatalf("unexpected panic envelope: %+v", body)
	}
	var access map[string]interface{}
	for _, line := range strings.Split(strings.TrimSpace(string(raw)), "\n") {
		_ = json.Unmarshal([]byte(line), &access)
		if access["app.log_type"] == "access" {
			break
		}
	}
	if access["error.stack_trace"] == nil || access["app.response_code"] != "CORE-9000" {
		t.Fatalf("panic access log missing stack trace or registry code: %#v", access)
	}
	ctxCleanup, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	_ = application.Lifecycle().Shutdown(ctxCleanup)
}
