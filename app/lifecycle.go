package app

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/yogayulanda/go-core/logger"
)

// Lifecycle manages graceful shutdown of application resources.
type Lifecycle struct {
	timeout time.Duration
	logger  logger.Logger

	mu    sync.Mutex
	hooks []func(ctx context.Context) error
}

// NewLifecycle creates lifecycle manager.
func NewLifecycle(timeout time.Duration, log logger.Logger) *Lifecycle {
	return &Lifecycle{
		timeout: timeout,
		logger:  log,
		hooks:   make([]func(ctx context.Context) error, 0),
	}
}

// Register adds a shutdown hook.
// Hooks are executed in reverse order (LIFO).
func (l *Lifecycle) Register(hook func(ctx context.Context) error) {
	if hook == nil {
		if l.logger != nil {
			l.logger.Warn(context.Background(), "ignored nil shutdown hook")
		}
		return
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	l.hooks = append(l.hooks, hook)
}

// Shutdown executes all registered hooks gracefully.
func (l *Lifecycle) Shutdown(ctx context.Context) error {
	if l.logger != nil {
		l.logger.Info(ctx, "starting graceful shutdown")
	}

	// Apply one global timeout budget if caller does not provide deadline.
	if _, hasDeadline := ctx.Deadline(); !hasDeadline && l.timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, l.timeout)
		defer cancel()
	}

	l.mu.Lock()
	hooks := make([]func(ctx context.Context) error, len(l.hooks))
	copy(hooks, l.hooks)
	l.mu.Unlock()

	var lastErr error

	// Execute in reverse order (LIFO)
	for i := len(hooks) - 1; i >= 0; i-- {
		if ctx.Err() != nil {
			if lastErr == nil {
				lastErr = ctx.Err()
			} else {
				lastErr = errors.Join(lastErr, ctx.Err())
			}
			break
		}

		err := func() (err error) {
			defer func() {
				if r := recover(); r != nil {
					err = fmt.Errorf("shutdown hook panic: %v", r)
				}
			}()
			return hooks[i](ctx)
		}()

		if err != nil {
			if lastErr == nil {
				lastErr = err
			} else {
				lastErr = errors.Join(lastErr, err)
			}
			if l.logger != nil {
				l.logger.Error(ctx, "shutdown hook failed",
					logger.Field{Key: "error", Value: err.Error()},
				)
			}
		}
	}

	if l.logger != nil {
		l.logger.Info(ctx, "graceful shutdown completed")
	}

	return lastErr
}
