package logger

import (
	"context"
	"strings"
	"time"

	gormlogger "gorm.io/gorm/logger"
)

type gormLogger struct {
	log       Logger
	threshold time.Duration
}

func NewGORMLogger(log Logger, slowThreshold time.Duration) gormlogger.Interface {
	if slowThreshold <= 0 {
		slowThreshold = time.Second
	}
	return &gormLogger{log: log, threshold: slowThreshold}
}

func (g *gormLogger) LogMode(gormlogger.LogLevel) gormlogger.Interface { return g }

func (g *gormLogger) Info(context.Context, string, ...interface{}) {}
func (g *gormLogger) Warn(context.Context, string, ...interface{}) {}
func (g *gormLogger) Error(ctx context.Context, message string, args ...interface{}) {
	if g.log == nil {
		return
	}
	g.log.Error(ctx, "database error", Field{Key: "app.log_type", Value: "db"}, Field{Key: "error.message", Value: message}, Field{Key: "app.metadata", Value: args})
}

func (g *gormLogger) Trace(ctx context.Context, startedAt time.Time, fc func() (string, int64), err error) {
	if g.log == nil {
		return
	}
	if err != nil && strings.Contains(strings.ToLower(err.Error()), "record not found") {
		return
	}
	duration := time.Since(startedAt)
	query, rows := fc()
	status := "success"
	errorCode := ""
	if err != nil {
		status = "failed"
		errorCode = "query_failed"
	}
	meta := map[string]interface{}{"rows_affected": rows}
	if err != nil {
		meta["error"] = err.Error()
	}
	g.log.LogDB(ctx, DBLog{Operation: "query", Query: query, Status: status, DurationMs: duration.Milliseconds(), ErrorCode: errorCode, Metadata: meta, Slow: err == nil && g.threshold > 0 && duration > g.threshold})
}
