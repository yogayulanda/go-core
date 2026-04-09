package resilience

import (
	"context"
	"fmt"
	"time"
)

// WithTimeout executes fn with a derived timeout context.
func WithTimeout(ctx context.Context, timeout time.Duration, fn func(ctx context.Context) error) error {
	return withTimeoutObserved(ctx, timeout, nil, fn)
}

func WithTimeoutObserved(
	ctx context.Context,
	timeout time.Duration,
	hook TimeoutHook,
	fn func(ctx context.Context) error,
) error {
	return withTimeoutObserved(ctx, timeout, hook, fn)
}

func withTimeoutObserved(
	ctx context.Context,
	timeout time.Duration,
	hook TimeoutHook,
	fn func(ctx context.Context) error,
) error {
	if fn == nil {
		return fmt.Errorf("resilience: fn is nil")
	}
	if timeout <= 0 {
		err := fn(ctx)
		if hook != nil && err == nil {
			hook(ctx, TimeoutEvent{Timeout: timeout, Status: "success"})
		}
		return err
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	err := fn(timeoutCtx)
	if hook == nil {
		return err
	}

	switch err {
	case nil:
		hook(ctx, TimeoutEvent{Timeout: timeout, Status: "success"})
	case context.DeadlineExceeded:
		hook(ctx, TimeoutEvent{Timeout: timeout, Status: "timeout", Err: err})
	case context.Canceled:
		hook(ctx, TimeoutEvent{Timeout: timeout, Status: "canceled", Err: err})
	}

	return err
}
