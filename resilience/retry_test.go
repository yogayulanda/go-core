package resilience

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/yogayulanda/go-core/logger"
)

func TestWithTimeout(t *testing.T) {
	ctx := context.Background()
	err := WithTimeout(ctx, 10*time.Millisecond, func(ctx context.Context) error {
		<-ctx.Done()
		return ctx.Err()
	})
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected deadline exceeded, got %v", err)
	}
}

func TestDo_SuccessAfterRetry(t *testing.T) {
	attempt := 0
	err := Do(context.Background(), RetryOptions{
		MaxAttempts: 3,
		BaseDelay:   time.Millisecond,
		MaxDelay:    2 * time.Millisecond,
		Jitter:      0,
		Retryable:   func(err error) bool { return true },
	}, func(context.Context) error {
		attempt++
		if attempt < 3 {
			return errors.New("temporary")
		}
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if attempt != 3 {
		t.Fatalf("expected 3 attempts, got %d", attempt)
	}
}

func TestDo_NoRetryWhenNotRetryable(t *testing.T) {
	attempt := 0
	err := Do(context.Background(), RetryOptions{
		MaxAttempts: 5,
		BaseDelay:   time.Millisecond,
		MaxDelay:    2 * time.Millisecond,
		Jitter:      0,
		Retryable:   func(err error) bool { return false },
	}, func(context.Context) error {
		attempt++
		return errors.New("fatal")
	})
	if err == nil {
		t.Fatalf("expected error")
	}
	if attempt != 1 {
		t.Fatalf("expected single attempt, got %d", attempt)
	}
}

func TestDo_ContextCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := Do(ctx, DefaultRetryOptions(), func(context.Context) error { return nil })
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context canceled, got %v", err)
	}
}

func TestDo_OnRetryHook_ReceivesRetryEvent(t *testing.T) {
	attempt := 0
	var events []RetryEvent

	err := Do(context.Background(), RetryOptions{
		MaxAttempts: 2,
		BaseDelay:   time.Millisecond,
		MaxDelay:    time.Millisecond,
		Jitter:      0,
		Retryable:   func(error) bool { return true },
		OnRetry: func(ctx context.Context, event RetryEvent) {
			events = append(events, event)
		},
	}, func(context.Context) error {
		attempt++
		if attempt == 1 {
			return context.DeadlineExceeded
		}
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(events) != 1 || events[0].Status != "retry_scheduled" {
		t.Fatalf("unexpected retry events: %+v", events)
	}
}

func TestWithTimeoutObserved_TimeoutHook_ReceivesTimeout(t *testing.T) {
	var events []TimeoutEvent

	err := WithTimeoutObserved(context.Background(), 10*time.Millisecond, func(ctx context.Context, event TimeoutEvent) {
		events = append(events, event)
	}, func(ctx context.Context) error {
		<-ctx.Done()
		return ctx.Err()
	})
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected deadline exceeded, got %v", err)
	}
	if len(events) != 1 || events[0].Status != "timeout" {
		t.Fatalf("unexpected timeout events: %+v", events)
	}
}

func TestRetryServiceLogHook_EmitsStructuredLog(t *testing.T) {
	log := &captureResilienceLogger{}
	hook := RetryServiceLogHook(log, "partner_call", map[string]interface{}{"dependency": "ledger"})

	hook(context.Background(), RetryEvent{
		Attempt:     1,
		MaxAttempts: 3,
		Delay:       50 * time.Millisecond,
		Status:      "retry_scheduled",
		Err:         context.DeadlineExceeded,
	})

	if len(log.serviceLogs) != 1 {
		t.Fatalf("expected 1 service log, got %d", len(log.serviceLogs))
	}
	entry := log.serviceLogs[0]
	if entry.Operation != "resilience_retry" || entry.Status != "retry_scheduled" {
		t.Fatalf("unexpected retry log: %+v", entry)
	}
}

type captureResilienceLogger struct {
	serviceLogs []logger.ServiceLog
}

func (l *captureResilienceLogger) Info(context.Context, string, ...logger.Field)         {}
func (l *captureResilienceLogger) Error(context.Context, string, ...logger.Field)        {}
func (l *captureResilienceLogger) Debug(context.Context, string, ...logger.Field)        {}
func (l *captureResilienceLogger) Warn(context.Context, string, ...logger.Field)         {}
func (l *captureResilienceLogger) LogDB(context.Context, logger.DBLog)                   {}
func (l *captureResilienceLogger) LogEvent(context.Context, logger.EventLog)             {}
func (l *captureResilienceLogger) LogTransaction(context.Context, logger.TransactionLog) {}
func (l *captureResilienceLogger) WithComponent(string) logger.Logger                    { return l }
func (l *captureResilienceLogger) LogService(_ context.Context, s logger.ServiceLog) {
	l.serviceLogs = append(l.serviceLogs, s)
}
