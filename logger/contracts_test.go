package logger

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/yogayulanda/go-core/observability"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func TestLogService_EmitsStructuredFields(t *testing.T) {
	log, buf := newJSONTestLogger(t)
	ctx := observability.WithRequestID(context.Background(), "req-123")

	log.LogService(ctx, ServiceLog{
		Operation:  "account_lookup",
		Status:     "success",
		DurationMs: 42,
		ErrorCode:  "NONE",
		Metadata: map[string]interface{}{
			"password": "secret123",
		},
	})

	entry := decodeLastJSONLog(t, buf.String())
	if entry["message"] != "service_log" {
		t.Fatalf("expected service_log message, got %v", entry["message"])
	}
	if entry["category"] != "service" {
		t.Fatalf("expected category service, got %v", entry["category"])
	}
	if entry["operation"] != "account_lookup" {
		t.Fatalf("expected operation account_lookup, got %v", entry["operation"])
	}
	if entry["status"] != "success" {
		t.Fatalf("expected status success, got %v", entry["status"])
	}
	if entry["duration_ms"] != float64(42) {
		t.Fatalf("expected duration_ms 42, got %v", entry["duration_ms"])
	}
	if entry["request_id"] != "req-123" {
		t.Fatalf("expected request_id req-123, got %v", entry["request_id"])
	}

	metadata, ok := entry["metadata"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected metadata object")
	}
	if metadata["password"] == "secret123" {
		t.Fatalf("expected metadata password to be masked")
	}
}

func TestLogDB_EmitsStructuredFields(t *testing.T) {
	log, buf := newJSONTestLogger(t)

	log.LogDB(context.Background(), DBLog{
		Operation:  "query",
		Query:      "SELECT * FROM users WHERE password = ?",
		DBName:     "primary",
		Status:     "failed",
		DurationMs: 15,
		ErrorCode:  "query_failed",
	})

	entry := decodeLastJSONLog(t, buf.String())
	if entry["message"] != "db_log" {
		t.Fatalf("expected db_log message, got %v", entry["message"])
	}
	if entry["category"] != "database" {
		t.Fatalf("expected category database, got %v", entry["category"])
	}
	if entry["db_name"] != "primary" {
		t.Fatalf("expected db_name primary, got %v", entry["db_name"])
	}
	if entry["query"] != "SELECT * FROM users WHERE password = ?" {
		t.Fatalf("expected query field to be present, got %v", entry["query"])
	}
}

func TestLogService_InjectsTraceID(t *testing.T) {
	log, buf := newJSONTestLogger(t)

	tp := sdktrace.NewTracerProvider()
	defer func() { _ = tp.Shutdown(context.Background()) }()
	tracer := tp.Tracer("logger-test")
	ctx, span := tracer.Start(context.Background(), "span")
	defer span.End()

	log.LogService(ctx, ServiceLog{
		Operation: "job_execute",
		Status:    "success",
	})

	entry := decodeLastJSONLog(t, buf.String())
	gotTraceID, _ := entry["trace_id"].(string)
	if gotTraceID == "" {
		t.Fatalf("expected trace_id")
	}

	wantTraceID := trace.SpanFromContext(ctx).SpanContext().TraceID().String()
	if gotTraceID != wantTraceID {
		t.Fatalf("expected trace_id %s, got %s", wantTraceID, gotTraceID)
	}
}

func TestLogTransaction_NoRegression_MessageAndCategory(t *testing.T) {
	log, buf := newJSONTestLogger(t)

	log.LogTransaction(context.Background(), TransactionLog{
		Operation:     "payment_process",
		TransactionID: "txn-1",
		UserID:        "user-1",
		Status:        "success",
		DurationMs:    99,
	})

	entry := decodeLastJSONLog(t, buf.String())
	if entry["message"] != "transaction_log" {
		t.Fatalf("expected transaction_log message, got %v", entry["message"])
	}
	if entry["category"] != "transaction" {
		t.Fatalf("expected category transaction, got %v", entry["category"])
	}
	if entry["transaction_id"] != "txn-1" {
		t.Fatalf("expected transaction_id txn-1, got %v", entry["transaction_id"])
	}
}

func newJSONTestLogger(t *testing.T) (*zapLogger, *strings.Builder) {
	t.Helper()

	buf := &strings.Builder{}
	encoderCfg := zapcore.EncoderConfig{
		TimeKey:       "timestamp",
		LevelKey:      "level",
		MessageKey:    "message",
		EncodeTime:    zapcore.ISO8601TimeEncoder,
		EncodeLevel:   zapcore.LowercaseLevelEncoder,
		EncodeCaller:  zapcore.ShortCallerEncoder,
		CallerKey:     "caller",
		StacktraceKey: "stacktrace",
	}

	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderCfg),
		zapcore.AddSync(builderSyncer{builder: buf}),
		zapcore.DebugLevel,
	)

	return &zapLogger{l: zap.New(core)}, buf
}

type builderSyncer struct {
	builder *strings.Builder
}

func (b builderSyncer) Write(p []byte) (int, error) {
	return b.builder.Write(p)
}

func (b builderSyncer) Sync() error {
	return nil
}

func decodeLastJSONLog(t *testing.T, raw string) map[string]interface{} {
	t.Helper()

	lines := strings.Split(strings.TrimSpace(raw), "\n")
	if len(lines) == 0 || strings.TrimSpace(lines[len(lines)-1]) == "" {
		t.Fatalf("expected log line, got %q", raw)
	}

	var out map[string]interface{}
	if err := json.Unmarshal([]byte(lines[len(lines)-1]), &out); err != nil {
		t.Fatalf("decode log json failed: %v", err)
	}
	return out
}
