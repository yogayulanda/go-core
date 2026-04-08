package logger

import "context"

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
}

func (z *zapLogger) LogDB(ctx context.Context, dbLog DBLog) {
	fields := []Field{
		{Key: "category", Value: "database"},
		{Key: "operation", Value: dbLog.Operation},
		{Key: "db_name", Value: dbLog.DBName},
		{Key: "status", Value: dbLog.Status},
		{Key: "duration_ms", Value: dbLog.DurationMs},
	}

	if dbLog.Query != "" {
		fields = append(fields, Field{Key: "query", Value: dbLog.Query})
	}

	if dbLog.ErrorCode != "" {
		fields = append(fields, Field{Key: "error_code", Value: dbLog.ErrorCode})
	}

	if dbLog.Metadata != nil {
		fields = append(fields, Field{Key: "metadata", Value: dbLog.Metadata})
	}

	z.Info(ctx, "db_log", fields...)
}
