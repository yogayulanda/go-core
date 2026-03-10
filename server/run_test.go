package server

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	coreapp "github.com/yogayulanda/go-core/app"
	"github.com/yogayulanda/go-core/config"
)

type testStartable struct {
	name  string
	start func() error
}

func (t testStartable) Name() string { return t.name }
func (t testStartable) Start() error { return t.start() }

func TestRun_ApplicationNil_ReturnError(t *testing.T) {
	err := Run(context.Background(), nil)
	if err == nil {
		t.Fatalf("expected error")
	}
	if !strings.Contains(err.Error(), "application is nil") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRun_ComponentError_ReturnError(t *testing.T) {
	application := newTestApp(t)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	err := Run(ctx, application, testStartable{
		name: "dummy_component",
		start: func() error {
			return errors.New("boom")
		},
	})

	if err == nil {
		t.Fatalf("expected error")
	}
	if !strings.Contains(err.Error(), "dummy_component failed") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRun_GracefulHTTPServerClosed_Ignored(t *testing.T) {
	application := newTestApp(t)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	err := Run(ctx, application, testStartable{
		name: "http_gateway",
		start: func() error {
			return http.ErrServerClosed
		},
	})
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

func TestRun_ComponentError_TriggersLifecycleShutdownHook(t *testing.T) {
	application := newTestApp(t)

	hookCalled := make(chan struct{}, 1)
	application.Lifecycle().Register(func(ctx context.Context) error {
		hookCalled <- struct{}{}
		return nil
	})

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	err := Run(ctx, application, testStartable{
		name: "dummy_component",
		start: func() error {
			return errors.New("boom")
		},
	})
	if err == nil {
		t.Fatalf("expected error")
	}

	select {
	case <-hookCalled:
	case <-time.After(500 * time.Millisecond):
		t.Fatalf("expected lifecycle shutdown hook to be called")
	}
}

func TestRun_ComponentError_ShutsDownOtherBlockingComponent(t *testing.T) {
	application := newTestApp(t)

	stopCh := make(chan struct{})
	application.Lifecycle().Register(func(ctx context.Context) error {
		close(stopCh)
		return nil
	})

	var blockingStarted atomic.Bool

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	err := Run(ctx, application,
		testStartable{
			name: "blocking_component",
			start: func() error {
				blockingStarted.Store(true)
				<-stopCh
				return http.ErrServerClosed
			},
		},
		testStartable{
			name: "failing_component",
			start: func() error {
				return errors.New("boom")
			},
		},
	)
	if err == nil {
		t.Fatalf("expected error")
	}
	if !blockingStarted.Load() {
		t.Fatalf("expected blocking component to start")
	}
	if !strings.Contains(err.Error(), "failing_component failed") {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Contains(err.Error(), "blocking_component failed") {
		t.Fatalf("blocking component should stop gracefully, got: %v", err)
	}
}

func TestRun_ComponentPanic_ReturnErrorAndShutdown(t *testing.T) {
	application := newTestApp(t)

	hookCalled := make(chan struct{}, 1)
	application.Lifecycle().Register(func(ctx context.Context) error {
		hookCalled <- struct{}{}
		return nil
	})

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	err := Run(ctx, application, testStartable{
		name: "panic_component",
		start: func() error {
			panic("boom")
		},
	})
	if err == nil {
		t.Fatalf("expected error")
	}
	if !strings.Contains(err.Error(), "panic_component failed") {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(err.Error(), "panic recovered: boom") {
		t.Fatalf("panic detail missing: %v", err)
	}

	select {
	case <-hookCalled:
	case <-time.After(500 * time.Millisecond):
		t.Fatalf("expected lifecycle shutdown hook to be called")
	}
}

func TestRun_ComponentEmptyName_UsesDefaultLabel(t *testing.T) {
	application := newTestApp(t)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	err := Run(ctx, application, testStartable{
		name: "   ",
		start: func() error {
			return errors.New("boom")
		},
	})
	if err == nil {
		t.Fatalf("expected error")
	}
	if !strings.Contains(err.Error(), "component failed: boom") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRun_ContextCancelled_BlockingComponent_ReturnsTimeout(t *testing.T) {
	application := newTestAppWithTimeout(t, 80*time.Millisecond)

	stopCh := make(chan struct{})
	defer close(stopCh)

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(20 * time.Millisecond)
		cancel()
	}()

	err := Run(ctx, application, testStartable{
		name: "blocking_component",
		start: func() error {
			<-stopCh
			return http.ErrServerClosed
		},
	})
	if err == nil {
		t.Fatalf("expected timeout error")
	}
	if !strings.Contains(err.Error(), "shutdown wait timeout") {
		t.Fatalf("expected shutdown wait timeout, got: %v", err)
	}
}

func TestRun_ComponentError_BlockingComponent_ReturnsJoinedTimeout(t *testing.T) {
	application := newTestAppWithTimeout(t, 80*time.Millisecond)

	stopCh := make(chan struct{})
	defer close(stopCh)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	err := Run(ctx, application,
		testStartable{
			name: "failing_component",
			start: func() error {
				return errors.New("boom")
			},
		},
		testStartable{
			name: "blocking_component",
			start: func() error {
				<-stopCh
				return http.ErrServerClosed
			},
		},
	)
	if err == nil {
		t.Fatalf("expected error")
	}
	if !strings.Contains(err.Error(), "failing_component failed: boom") {
		t.Fatalf("expected failing component error, got: %v", err)
	}
	if !strings.Contains(err.Error(), "shutdown wait timeout") {
		t.Fatalf("expected shutdown wait timeout, got: %v", err)
	}
}

func newTestApp(t *testing.T) *coreapp.App {
	t.Helper()
	return newTestAppWithTimeout(t, time.Second)
}

func newTestAppWithTimeout(t *testing.T, shutdownTimeout time.Duration) *coreapp.App {
	t.Helper()

	cfg := &config.Config{
		App: config.AppConfig{
			ServiceName:     "server-run-test",
			Environment:     "test",
			LogLevel:        "error",
			ShutdownTimeout: shutdownTimeout,
		},
		Databases: map[string]config.DBConfig{},
	}

	application, err := coreapp.New(context.Background(), cfg)
	if err != nil {
		t.Fatalf("init app failed: %v", err)
	}
	return application
}
