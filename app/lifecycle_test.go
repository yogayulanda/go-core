package app

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/yogayulanda/go-core/logger"
)

func TestLifecycleShutdown_GlobalTimeoutBudgetRespected(t *testing.T) {
	log := newLifecycleTestLogger(t)
	lc := NewLifecycle(100*time.Millisecond, log)

	sleepHook := func(ctx context.Context) error {
		select {
		case <-time.After(60 * time.Millisecond):
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	lc.Register(sleepHook)
	lc.Register(sleepHook)

	start := time.Now()
	err := lc.Shutdown(context.Background())
	elapsed := time.Since(start)

	if err == nil {
		t.Fatalf("expected timeout error")
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected deadline exceeded, got: %v", err)
	}
	if elapsed > 180*time.Millisecond {
		t.Fatalf("shutdown took too long, elapsed=%v", elapsed)
	}
}

func TestLifecycleShutdown_CallerDeadlineTakesPrecedence(t *testing.T) {
	log := newLifecycleTestLogger(t)
	lc := NewLifecycle(5*time.Second, log)

	lc.Register(func(ctx context.Context) error {
		select {
		case <-time.After(200 * time.Millisecond):
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	})

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	start := time.Now()
	err := lc.Shutdown(ctx)
	elapsed := time.Since(start)

	if err == nil {
		t.Fatalf("expected timeout error")
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected deadline exceeded, got: %v", err)
	}
	if elapsed > 120*time.Millisecond {
		t.Fatalf("caller deadline not respected, elapsed=%v", elapsed)
	}
}

func TestLifecycleShutdown_JoinHookErrors(t *testing.T) {
	log := newLifecycleTestLogger(t)
	lc := NewLifecycle(time.Second, log)

	lc.Register(func(ctx context.Context) error { return errors.New("err-A") })
	lc.Register(func(ctx context.Context) error { return errors.New("err-B") })

	err := lc.Shutdown(context.Background())
	if err == nil {
		t.Fatalf("expected error")
	}

	msg := err.Error()
	if !strings.Contains(msg, "err-A") || !strings.Contains(msg, "err-B") {
		t.Fatalf("expected joined errors, got: %v", err)
	}
}

func TestLifecycleRegister_NilHookIgnored(t *testing.T) {
	log := newLifecycleTestLogger(t)
	lc := NewLifecycle(time.Second, log)

	lc.Register(nil)

	if err := lc.Shutdown(context.Background()); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

func TestLifecycleShutdown_PanicHook_ConvertedToErrorAndContinues(t *testing.T) {
	log := newLifecycleTestLogger(t)
	lc := NewLifecycle(time.Second, log)

	lc.Register(func(ctx context.Context) error { return errors.New("err-A") })
	lc.Register(func(ctx context.Context) error { panic("boom") })

	err := lc.Shutdown(context.Background())
	if err == nil {
		t.Fatalf("expected error")
	}

	msg := err.Error()
	if !strings.Contains(msg, "shutdown hook panic: boom") {
		t.Fatalf("expected panic converted to error, got: %v", err)
	}
	if !strings.Contains(msg, "err-A") {
		t.Fatalf("expected shutdown to continue to remaining hook, got: %v", err)
	}
}

func TestLifecycleShutdown_NilLogger_NoPanic(t *testing.T) {
	lc := NewLifecycle(time.Second, nil)

	lc.Register(nil)
	lc.Register(func(ctx context.Context) error { return nil })

	if err := lc.Shutdown(context.Background()); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

func newLifecycleTestLogger(t *testing.T) logger.Logger {
	t.Helper()
	log, err := logger.New("lifecycle-test", "error")
	if err != nil {
		t.Fatalf("init logger failed: %v", err)
	}
	return log
}
