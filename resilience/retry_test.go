package resilience

import (
	"context"
	"errors"
	"testing"
	"time"
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
