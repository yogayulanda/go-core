package messaging

import (
	"context"
	"time"
)

func executeWithRetry(
	ctx context.Context,
	enabled bool,
	maxRetries int,
	delay time.Duration,
	exec func(context.Context) error,
) error {
	if !enabled {
		return exec(ctx)
	}

	if maxRetries < 0 {
		maxRetries = 0
	}
	if delay < 0 {
		delay = 0
	}

	var err error
	for attempt := 0; attempt <= maxRetries; attempt++ {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		err = exec(ctx)
		if err == nil {
			return nil
		}

		if attempt == maxRetries {
			break
		}
		if waitErr := waitRetryDelay(ctx, delay); waitErr != nil {
			return waitErr
		}
	}

	return err
}

func waitRetryDelay(ctx context.Context, delay time.Duration) error {
	if delay <= 0 {
		return nil
	}

	timer := time.NewTimer(delay)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
