package logger

import "context"

// ServiceLog represents the standard structured log contract
// for normal technical service flow.
//
// Use it for:
// - request/use-case execution
// - service orchestration milestones
// - internal technical flow status
//
// Top-level fields are intentionally small and stable.
// Additional service-specific context should go into Metadata.
type ServiceLog struct {
	Operation  string                 // stable service flow or step name
	Status     string                 // stable status, e.g. "success", "failed", "started"
	DurationMs int64                  // execution duration in milliseconds
	ErrorCode  string                 // optional stable classification for monitoring
	Metadata   map[string]interface{} // additional structured service-specific attributes
}

func (z *zapLogger) LogService(ctx context.Context, svc ServiceLog) {
	fields := []Field{
		{Key: "category", Value: "service"},
		{Key: "operation", Value: svc.Operation},
		{Key: "status", Value: svc.Status},
		{Key: "duration_ms", Value: svc.DurationMs},
	}

	if svc.ErrorCode != "" {
		fields = append(fields, Field{Key: "error_code", Value: svc.ErrorCode})
	}

	if svc.Metadata != nil {
		fields = append(fields, Field{Key: "metadata", Value: svc.Metadata})
	}

	z.Info(ctx, "service_log", fields...)
}
