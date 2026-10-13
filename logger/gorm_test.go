package logger

import (
	"context"
	"errors"
	"testing"
	"time"
)

type dbLogRecorder struct {
	entries []DBLog
}

func (r *dbLogRecorder) Info(context.Context, string, ...Field)         {}
func (r *dbLogRecorder) Error(context.Context, string, ...Field)        {}
func (r *dbLogRecorder) Debug(context.Context, string, ...Field)        {}
func (r *dbLogRecorder) Warn(context.Context, string, ...Field)         {}
func (r *dbLogRecorder) LogService(context.Context, ServiceLog)         {}
func (r *dbLogRecorder) LogDB(_ context.Context, entry DBLog)           { r.entries = append(r.entries, entry) }
func (r *dbLogRecorder) LogEvent(context.Context, EventLog)             {}
func (r *dbLogRecorder) LogTransaction(context.Context, TransactionLog) {}
func (r *dbLogRecorder) WithComponent(string) Logger                    { return r }

func TestGORMLogger_IgnoresRecordNotFound(t *testing.T) {
	recorder := &dbLogRecorder{}
	adapter := NewGORMLogger(recorder, time.Second)
	adapter.Trace(context.Background(), time.Now(), func() (string, int64) { return "SELECT 1", 0 }, errors.New("record not found"))
	if len(recorder.entries) != 0 {
		t.Fatalf("record not found should be ignored, got %+v", recorder.entries)
	}
}

func TestGORMLogger_ReportsSlowQuery(t *testing.T) {
	recorder := &dbLogRecorder{}
	adapter := NewGORMLogger(recorder, 10*time.Millisecond)
	adapter.Trace(context.Background(), time.Now().Add(-20*time.Millisecond), func() (string, int64) { return "SELECT 1", 1 }, nil)
	if len(recorder.entries) != 1 || recorder.entries[0].DurationMs < 10 || !recorder.entries[0].Slow {
		t.Fatalf("expected one slow query log, got %+v", recorder.entries)
	}
}
