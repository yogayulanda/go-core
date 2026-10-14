package logger

import (
	"context"
	"strings"
	"testing"
)

type textLogRecorder struct {
	warnings []string
	errors   []string
}

func (r *textLogRecorder) Info(context.Context, string, ...Field) {}
func (r *textLogRecorder) Error(_ context.Context, message string, _ ...Field) {
	r.errors = append(r.errors, message)
}
func (r *textLogRecorder) Debug(context.Context, string, ...Field) {}
func (r *textLogRecorder) Warn(_ context.Context, message string, _ ...Field) {
	r.warnings = append(r.warnings, message)
}
func (*textLogRecorder) LogService(context.Context, ServiceLog)         {}
func (*textLogRecorder) LogDB(context.Context, DBLog)                   {}
func (*textLogRecorder) LogEvent(context.Context, EventLog)             {}
func (*textLogRecorder) LogTransaction(context.Context, TransactionLog) {}
func (r *textLogRecorder) WithComponent(string) Logger                  { return r }

func TestGRPCLogWriterFiltersRoutineAndRoutesWarningsErrors(t *testing.T) {
	recorder := &textLogRecorder{}
	warningWriter := grpcLogWriter{log: recorder, level: "warn"}
	errorWriter := grpcLogWriter{log: recorder, level: "error"}
	_, _ = warningWriter.Write([]byte("transport is closing\n"))
	_, _ = warningWriter.Write([]byte("client disconnected: EOF\n"))
	_, _ = warningWriter.Write([]byte("keepalive warning\n"))
	_, _ = errorWriter.Write([]byte("transport failure\n"))
	if len(recorder.warnings) != 1 || !strings.Contains(recorder.warnings[0], "keepalive warning") {
		t.Fatalf("unexpected warning routing: %#v", recorder.warnings)
	}
	if len(recorder.errors) != 1 || !strings.Contains(recorder.errors[0], "transport failure") {
		t.Fatalf("unexpected error routing: %#v", recorder.errors)
	}
}
