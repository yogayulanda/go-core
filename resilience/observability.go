package resilience

import (
	"context"
	"time"

	"github.com/yogayulanda/go-core/logger"
)

type RetryEvent struct {
	Attempt     int
	MaxAttempts int
	Delay       time.Duration
	Status      string
	Err         error
}

type TimeoutEvent struct {
	Timeout time.Duration
	Status  string
	Err     error
}

type RetryHook func(ctx context.Context, event RetryEvent)
type TimeoutHook func(ctx context.Context, event TimeoutEvent)

func RetryServiceLogHook(log logger.Logger, targetOperation string, metadata map[string]interface{}) RetryHook {
	return func(ctx context.Context, event RetryEvent) {
		if log == nil {
			return
		}
		fields := map[string]interface{}{
			"target_operation": targetOperation,
			"attempt":          event.Attempt,
			"max_attempts":     event.MaxAttempts,
			"delay_ms":         event.Delay.Milliseconds(),
		}
		for k, v := range metadata {
			fields[k] = v
		}
		if event.Err != nil {
			fields["error"] = event.Err.Error()
		}
		log.LogService(ctx, logger.ServiceLog{
			Operation: "resilience_retry",
			Status:    event.Status,
			ErrorCode: retryEventErrorCode(event),
			Metadata:  fields,
		})
	}
}

func TimeoutServiceLogHook(log logger.Logger, targetOperation string, metadata map[string]interface{}) TimeoutHook {
	return func(ctx context.Context, event TimeoutEvent) {
		if log == nil {
			return
		}
		fields := map[string]interface{}{
			"target_operation": targetOperation,
			"timeout_ms":       event.Timeout.Milliseconds(),
		}
		for k, v := range metadata {
			fields[k] = v
		}
		if event.Err != nil {
			fields["error"] = event.Err.Error()
		}
		log.LogService(ctx, logger.ServiceLog{
			Operation: "resilience_timeout",
			Status:    event.Status,
			ErrorCode: timeoutEventErrorCode(event),
			Metadata:  fields,
		})
	}
}

func retryEventErrorCode(event RetryEvent) string {
	switch event.Status {
	case "retry_scheduled":
		return "retryable_error"
	case "stopped":
		return "not_retryable"
	case "canceled":
		return "context_canceled"
	default:
		return ""
	}
}

func timeoutEventErrorCode(event TimeoutEvent) string {
	switch event.Status {
	case "timeout":
		return "deadline_exceeded"
	case "canceled":
		return "context_canceled"
	default:
		return ""
	}
}
