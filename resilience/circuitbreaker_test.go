package resilience

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/sony/gobreaker/v2"
)

func TestCircuitBreaker(t *testing.T) {
	opts := DefaultCircuitBreakerOptions("test-cb")
	opts.Timeout = 50 * time.Millisecond
	opts.ReadyToTrip = func(counts gobreaker.Counts) bool {
		return counts.ConsecutiveFailures >= 2 // trip on 2 failures
	}

	cb := NewCircuitBreaker(opts)

	// Step 1: Success calls should work
	err := cb.Do(context.Background(), func(ctx context.Context) error {
		return nil
	})
	if err != nil {
		t.Fatalf("expected nil, got %v", err)
	}

	// Step 2: Failed calls
	myErr := errors.New("boom")
	_ = cb.Do(context.Background(), func(ctx context.Context) error {
		return myErr
	}) // 1st failure
	_ = cb.Do(context.Background(), func(ctx context.Context) error {
		return myErr
	}) // 2nd failure -> Trips Open

	// Step 3: Now should fail fast
	err = cb.Do(context.Background(), func(ctx context.Context) error {
		return nil // won't execute
	})
	if !errors.Is(err, ErrCircuitOpen) {
		t.Fatalf("expected ErrCircuitOpen, got %v", err)
	}
}
