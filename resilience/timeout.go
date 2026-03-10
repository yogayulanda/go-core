package resilience

import (
	"context"
	"fmt"
	"time"
)

// WithTimeout executes fn with a derived timeout context.
func WithTimeout(ctx context.Context, timeout time.Duration, fn func(ctx context.Context) error) error {
	if fn == nil {
		return fmt.Errorf("resilience: fn is nil")
	}
	if timeout <= 0 {
		return fn(ctx)
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	return fn(timeoutCtx)
}
