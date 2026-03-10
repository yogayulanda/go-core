package grpc

import (
	"context"
	"errors"
	"testing"
	"time"
)

type fakeStopper struct {
	gracefulCh chan struct{}
	stopCalled bool
}

func (f *fakeStopper) GracefulStop() {
	if f.gracefulCh != nil {
		<-f.gracefulCh
	}
}

func (f *fakeStopper) Stop() {
	f.stopCalled = true
	if f.gracefulCh != nil {
		select {
		case <-f.gracefulCh:
		default:
			close(f.gracefulCh)
		}
	}
}

func TestGracefulStopWithTimeout_GracefulPath_ReturnNil(t *testing.T) {
	st := &fakeStopper{
		gracefulCh: make(chan struct{}),
	}

	go func() {
		time.Sleep(20 * time.Millisecond)
		close(st.gracefulCh)
	}()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	err := gracefulStopWithTimeout(ctx, st)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if st.stopCalled {
		t.Fatalf("stop must not be called on graceful path")
	}
}

func TestGracefulStopWithTimeout_TimeoutPath_ForceStop(t *testing.T) {
	st := &fakeStopper{
		gracefulCh: make(chan struct{}),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	err := gracefulStopWithTimeout(ctx, st)
	if err == nil {
		t.Fatalf("expected timeout error")
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected deadline exceeded, got: %v", err)
	}
	if !st.stopCalled {
		t.Fatalf("stop must be called on timeout path")
	}
}
