package logger

import (
	"context"
	"fmt"
	"strings"
	"time"
)

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
	Operation         string                 // stable service flow or step name
	Status            string                 // stable status, e.g. "success", "failed", "started"
	DurationMs        int64                  // execution duration in milliseconds
	ErrorCode         string                 // optional stable classification for monitoring
	Metadata          map[string]interface{} // additional structured service-specific attributes
	LogType           string
	Class             string
	Route             string
	Method            string
	StatusCode        int
	ResponseCode      string
	ErrorMessage      string
	ErrorStackTrace   string
	UnregisteredCode  string
	RequestIDReplaced bool
	Slow              bool
}

func (z *zapLogger) LogService(ctx context.Context, svc ServiceLog) {
	if svc.LogType == "access" {
		z.logAccess(ctx, svc)
		return
	}
	logType := svc.LogType
	if logType == "" {
		logType = "service"
	}
	fields := []Field{
		{Key: "app.log_type", Value: logType},
		{Key: "app.operation", Value: svc.Operation},
		{Key: "app.status", Value: svc.Status},
		{Key: "app.duration_ms", Value: svc.DurationMs},
	}

	if svc.ErrorCode != "" {
		fields = append(fields, Field{Key: "error.code", Value: svc.ErrorCode})
	}

	if svc.Metadata != nil {
		fields = append(fields, Field{Key: "app.metadata", Value: svc.Metadata})
	}

	z.Info(ctx, "service "+svc.Operation+" "+svc.Status, fields...)
}

func (z *zapLogger) logAccess(ctx context.Context, access ServiceLog) {
	baseMessage := fmt.Sprintf("%s %s %d %s %dms", access.Method, access.Route, access.StatusCode, access.ResponseCode, access.DurationMs)
	message := baseMessage
	level := 1
	switch access.Class {
	case "UNKNOWN":
		message = "UNKNOWN " + baseMessage
		level = 2
	case "ERROR":
		message = "ERROR " + baseMessage
		level = 3
	}
	fields := []Field{
		{Key: "app.log_type", Value: "access"},
		{Key: "app.route", Value: access.Route},
		{Key: "app.method", Value: access.Method},
		{Key: "app.status_code", Value: access.StatusCode},
		{Key: "app.response_code", Value: access.ResponseCode},
		{Key: "app.class", Value: access.Class},
		{Key: "app.duration_ms", Value: access.DurationMs},
	}
	if access.UnregisteredCode != "" {
		fields = append(fields, Field{Key: "app.unregistered_code", Value: access.UnregisteredCode})
		level = 2
	}
	if access.RequestIDReplaced {
		fields = append(fields, Field{Key: "app.request_id_replaced", Value: true})
		level = 2
	}
	slow := access.Slow || (z.config.SlowRequestThreshold > 0 && time.Duration(access.DurationMs)*time.Millisecond > z.config.SlowRequestThreshold)
	if slow {
		if access.Class == "ERROR" || access.Class == "UNKNOWN" {
			message = strings.TrimSuffix(message, baseMessage) + "SLOW " + baseMessage
		} else {
			message = "SLOW " + baseMessage
		}
		fields = append(fields, Field{Key: "app.slow", Value: true})
		if access.Class != "ERROR" {
			level = 2
		}
	}
	if access.ErrorMessage != "" {
		fields = append(fields, Field{Key: "error.message", Value: access.ErrorMessage})
	}
	if access.ErrorStackTrace != "" {
		fields = append(fields, Field{Key: "error.stack_trace", Value: access.ErrorStackTrace})
	}
	switch level {
	case 3:
		z.Error(ctx, message, fields...)
	case 2:
		z.Warn(ctx, message, fields...)
	default:
		z.Info(ctx, message, fields...)
	}
}
