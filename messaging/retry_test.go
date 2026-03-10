package messaging

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestExecuteWithRetry_SuccessAfterRetry(t *testing.T) {
	attempt := 0
	err := executeWithRetry(
		context.Background(),
		true,
		2,
		0,
		func(context.Context) error {
			attempt++
			if attempt < 2 {
				return errors.New("temporary")
			}
			return nil
		},
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if attempt != 2 {
		t.Fatalf("expected 2 attempts, got %d", attempt)
	}
}

func TestExecuteWithRetry_ContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := executeWithRetry(
		ctx,
		true,
		2,
		time.Millisecond,
		func(context.Context) error {
			return errors.New("will not run")
		},
	)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context canceled, got: %v", err)
	}
}
