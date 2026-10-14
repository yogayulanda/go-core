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
	if entry["message"] != "service account_lookup success" {
		t.Fatalf("unexpected service message, got %v", entry["message"])
	}
	if entry["app.log_type"] != "service" {
		t.Fatalf("expected app.log_type service, got %v", entry["app.log_type"])
	}
	if entry["app.operation"] != "account_lookup" {
		t.Fatalf("expected app.operation account_lookup, got %v", entry["app.operation"])
	}
	if entry["app.status"] != "success" {
		t.Fatalf("expected app.status success, got %v", entry["app.status"])
	}
	if entry["app.duration_ms"] != float64(42) {
		t.Fatalf("expected app.duration_ms 42, got %v", entry["app.duration_ms"])
	}
	if entry["labels.request_id"] != "req-123" {
		t.Fatalf("expected labels.request_id req-123, got %v", entry["labels.request_id"])
	}

	metadata, ok := entry["app.metadata"].(map[string]interface{})
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
	if entry["message"] != "db query" {
		t.Fatalf("unexpected database message, got %v", entry["message"])
	}
	if entry["app.log_type"] != "db" {
		t.Fatalf("expected app.log_type db, got %v", entry["app.log_type"])
	}
	if entry["app.db_name"] != "primary" {
		t.Fatalf("expected app.db_name primary, got %v", entry["app.db_name"])
	}
	if entry["app.query"] != "SELECT * FROM users WHERE password = ?" {
		t.Fatalf("expected app.query field to be present, got %v", entry["app.query"])
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
	gotTraceID, _ := entry["trace.id"].(string)
	if gotTraceID == "" {
		t.Fatalf("expected trace.id")
	}

	wantTraceID := trace.SpanFromContext(ctx).SpanContext().TraceID().String()
	if gotTraceID != wantTraceID {
		t.Fatalf("expected trace.id %s, got %s", wantTraceID, gotTraceID)
	}
}

func TestLogTransaction_NoRegression_MessageAndCategory(t *testing.T) {
	log, buf := newJSONTestLogger(t)
	ctx := observability.WithGTID(context.Background(), "txn-1")

	log.LogTransaction(ctx, TransactionLog{
		Feature:      "payment",
		Product:      "transfer",
		Status:       "success",
		ResponseCode: "CORE-0000",
		DurationMs:   99,
	})

	entry := decodeLastJSONLog(t, buf.String())
	if entry["message"] != "transaction payment txn-1  -> success CORE-0000 99ms" {
		t.Fatalf("unexpected transaction message: %v", entry["message"])
	}
	if entry["app.log_type"] != "transaction" {
		t.Fatalf("expected transaction log type, got %v", entry["app.log_type"])
	}
	if entry["labels.gtid"] != "txn-1" {
		t.Fatalf("expected labels.gtid txn-1, got %v", entry["labels.gtid"])
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
