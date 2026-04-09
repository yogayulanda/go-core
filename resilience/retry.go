package resilience

import (
	"context"
	cryptorand "crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"time"
)

type RetryOptions struct {
	MaxAttempts int
	BaseDelay   time.Duration
	MaxDelay    time.Duration
	Jitter      time.Duration
	Retryable   func(err error) bool
	OnRetry     RetryHook
}

func DefaultRetryOptions() RetryOptions {
	return RetryOptions{
		MaxAttempts: 1,
		BaseDelay:   200 * time.Millisecond,
		MaxDelay:    2 * time.Second,
		Jitter:      100 * time.Millisecond,
		Retryable:   IsTransientError,
	}
}

func Do(
	ctx context.Context,
	opts RetryOptions,
	fn func(ctx context.Context) error,
) error {
	if fn == nil {
		return fmt.Errorf("resilience: fn is nil")
	}

	normalizeRetryOptions(&opts)

	var lastErr error
	for attempt := 1; attempt <= opts.MaxAttempts; attempt++ {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		err := fn(ctx)
		if err == nil {
			return nil
		}
		lastErr = err

		if attempt == opts.MaxAttempts {
			break
		}
		if !opts.Retryable(err) {
			if opts.OnRetry != nil {
				opts.OnRetry(ctx, RetryEvent{
					Attempt:     attempt,
					MaxAttempts: opts.MaxAttempts,
					Status:      "stopped",
					Err:         err,
				})
			}
			return err
		}

		delay := computeBackoff(attempt, opts)
		if opts.OnRetry != nil {
			opts.OnRetry(ctx, RetryEvent{
				Attempt:     attempt,
				MaxAttempts: opts.MaxAttempts,
				Delay:       delay,
				Status:      "retry_scheduled",
				Err:         err,
			})
		}
		select {
		case <-ctx.Done():
			if opts.OnRetry != nil {
				opts.OnRetry(ctx, RetryEvent{
					Attempt:     attempt,
					MaxAttempts: opts.MaxAttempts,
					Status:      "canceled",
					Err:         ctx.Err(),
				})
			}
			return ctx.Err()
		case <-time.After(delay):
		}
	}

	if lastErr == nil {
		return errors.New("resilience: retry completed with unknown error")
	}
	return lastErr
}

func IsTransientError(err error) bool {
	if err == nil {
		return false
	}
	return errors.Is(err, context.DeadlineExceeded)
}

func normalizeRetryOptions(opts *RetryOptions) {
	if opts.MaxAttempts <= 0 {
		opts.MaxAttempts = 1
	}
	if opts.BaseDelay <= 0 {
		opts.BaseDelay = 200 * time.Millisecond
	}
	if opts.MaxDelay <= 0 {
		opts.MaxDelay = 2 * time.Second
	}
	if opts.Jitter < 0 {
		opts.Jitter = 0
	}
	if opts.Retryable == nil {
		opts.Retryable = IsTransientError
	}
}

func computeBackoff(attempt int, opts RetryOptions) time.Duration {
	if attempt <= 1 {
		return applyJitter(opts.BaseDelay, opts.Jitter)
	}

	delay := opts.BaseDelay
	for i := 1; i < attempt; i++ {
		delay *= 2
		if delay > opts.MaxDelay {
			delay = opts.MaxDelay
			break
		}
	}

	return applyJitter(delay, opts.Jitter)
}

func applyJitter(base, jitter time.Duration) time.Duration {
	if jitter <= 0 {
		return base
	}
	max := big.NewInt(int64(jitter) + 1)
	n, err := cryptorand.Int(cryptorand.Reader, max)
	if err != nil {
		return base
	}
	extra := time.Duration(n.Int64())
	return base + extra
}
