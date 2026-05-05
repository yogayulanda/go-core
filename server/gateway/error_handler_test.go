package gateway

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/yogayulanda/go-core/config"
	coreErrors "github.com/yogayulanda/go-core/errors"
	"github.com/yogayulanda/go-core/observability"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestCustomErrorHandler_ValidationError_CompactBodyWithDetails(t *testing.T) {
	application := newTestApp(t, &config.Config{
		App: config.AppConfig{
			ServiceName:     "gateway-error-handler-test",
			Environment:     "test",
			LogLevel:        "error",
			ShutdownTimeout: time.Second,
		},
		Databases: map[string]config.DBConfig{},
	})

	handler := customErrorHandler(application)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/example", nil)
	ctx := observability.WithRequestID(req.Context(), "req-123")

	handler(
		ctx,
		runtime.NewServeMux(),
		nil,
		rec,
		req,
		coreErrors.ToGRPC(coreErrors.Validation("invalid request", coreErrors.Detail{Field: "user_id", Reason: "required"})),
	)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
	if got := rec.Header().Get("x-request-id"); got != "req-123" {
		t.Fatalf("unexpected x-request-id: %s", got)
	}

	var body coreErrors.ErrorResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal response failed: %v", err)
	}
	if body.Code != string(coreErrors.CodeInvalidRequest) {
		t.Fatalf("unexpected code: %s", body.Code)
	}
	if body.Message != "invalid request" {
		t.Fatalf("unexpected message: %s", body.Message)
	}
	if len(body.Details) != 1 {
		t.Fatalf("expected 1 detail, got %d", len(body.Details))
	}
	if body.Success != false {
		t.Fatalf("expected success: false")
	}
	if body.TraceID != "req-123" {
		t.Fatalf("expected trace_id req-123, got %s", body.TraceID)
	}
	if body.Timestamp == "" {
		t.Fatalf("expected timestamp, got empty")
	}
}

func TestCustomErrorHandler_UnknownGRPCError_IsSanitized(t *testing.T) {
	application := newTestApp(t, &config.Config{
		App: config.AppConfig{
			ServiceName:     "gateway-error-handler-test",
			Environment:     "test",
			LogLevel:        "error",
			ShutdownTimeout: time.Second,
		},
		Databases: map[string]config.DBConfig{},
	})

	handler := customErrorHandler(application)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/example", nil)

	handler(
		context.Background(),
		runtime.NewServeMux(),
		nil,
		rec,
		req,
		status.Error(codes.Internal, "raw internal"),
	)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rec.Code)
	}

	var body coreErrors.ErrorResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal response failed: %v", err)
	}
	if body.Code != string(coreErrors.CodeInternal) {
		t.Fatalf("unexpected code: %s", body.Code)
	}
	if body.Message != "internal server error" {
		t.Fatalf("unexpected message: %s", body.Message)
	}
	if len(body.Details) != 0 {
		t.Fatalf("expected no details, got %v", body.Details)
	}
}

func TestCustomErrorHandler_DirectAppError_UsesCanonicalContract(t *testing.T) {
	application := newTestApp(t, &config.Config{
		App: config.AppConfig{
			ServiceName:     "gateway-error-handler-test",
			Environment:     "test",
			LogLevel:        "error",
			ShutdownTimeout: time.Second,
		},
		Databases: map[string]config.DBConfig{},
	})

	handler := customErrorHandler(application)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/example", nil)

	handler(
		context.Background(),
		runtime.NewServeMux(),
		nil,
		rec,
		req,
		coreErrors.New(coreErrors.CodeSessionExpired, ""),
	)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}

	var body coreErrors.ErrorResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal response failed: %v", err)
	}
	if body.Code != string(coreErrors.CodeSessionExpired) {
		t.Fatalf("unexpected code: %s", body.Code)
	}
	if body.Message != "session expired" {
		t.Fatalf("unexpected message: %s", body.Message)
	}
}
