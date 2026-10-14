package logger

import (
	"context"
	"fmt"
	"time"
)

// DBLog represents the standard structured log contract
// for database operational and query-related logging.
//
// Use it for:
// - connect and ping lifecycle
// - retry/timeout/failure reporting
// - slow query reporting
// - pool or driver-related operational events
//
// Query is optional and should contain sanitized or controlled text only.
// Callers must avoid logging raw secrets or unsafe parameter payloads.
type DBLog struct {
	Operation  string                 // stable DB operation, e.g. "connect", "ping", "query"
	Query      string                 // optional sanitized query text
	DBName     string                 // database name or configured target when available
	Status     string                 // stable status, e.g. "success", "failed", "timeout"
	DurationMs int64                  // execution duration in milliseconds
	ErrorCode  string                 // optional stable classification for monitoring
	Metadata   map[string]interface{} // additional structured database-specific attributes
	Slow       bool
}

func (z *zapLogger) LogDB(ctx context.Context, dbLog DBLog) {
	fields := []Field{
		{Key: "app.log_type", Value: "db"},
		{Key: "app.operation", Value: dbLog.Operation},
		{Key: "app.db_name", Value: dbLog.DBName},
		{Key: "app.status", Value: dbLog.Status},
		{Key: "app.duration_ms", Value: dbLog.DurationMs},
	}

	if dbLog.Query != "" {
		fields = append(fields, Field{Key: "app.query", Value: dbLog.Query})
	}

	if dbLog.ErrorCode != "" {
		fields = append(fields, Field{Key: "error.code", Value: dbLog.ErrorCode})
	}

	if dbLog.Metadata != nil {
		fields = append(fields, Field{Key: "app.metadata", Value: dbLog.Metadata})
	}
	if dbLog.Operation == "query" && dbLog.Status == "not_found" {
		return
	}
	if dbLog.Status == "failed" || dbLog.Status == "error" {
		z.Error(ctx, "db "+dbLog.Operation, fields...)
		return
	}
	if dbLog.Slow || (z.config.SlowQueryThreshold > 0 && time.Duration(dbLog.DurationMs)*time.Millisecond > z.config.SlowQueryThreshold) {
		fields = append(fields, Field{Key: "app.slow", Value: true})
		z.Warn(ctx, fmt.Sprintf("SLOW SQL %d %s", dbLog.DurationMs, dbLog.Query), fields...)
		return
	}
	z.Info(ctx, "db "+dbLog.Operation, fields...)
}
