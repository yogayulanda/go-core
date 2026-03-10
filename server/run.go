package server

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/yogayulanda/go-core/app"
)

type Startable interface {
	Name() string
	Start() error
}

type runResult struct {
	name string
	err  error
}

// Run starts infrastructure components and binds them to app lifecycle shutdown.
//
// Behavior:
// - starts all components concurrently
// - starts app lifecycle loop
// - if any component/app returns non-graceful error, context is cancelled and shutdown is triggered
// - graceful stop errors from HTTP/gRPC servers are ignored
func Run(ctx context.Context, application *app.App, components ...Startable) error {
	if application == nil {
		return fmt.Errorf("application is nil")
	}

	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	total := 1 // app.Start
	for _, c := range components {
		if c != nil {
			total++
		}
	}

	resultCh := make(chan runResult, total)
	waitTimeout := resolveShutdownWaitTimeout(application)

	for _, component := range components {
		if component == nil {
			continue
		}

		c := component
		go func() {
			name := normalizeComponentName(c.Name())
			err := safeStart(c.Start)
			if isExpectedServeStop(err) {
				err = nil
			}
			resultCh <- runResult{name: name, err: err}
		}()
	}

	go func() {
		resultCh <- runResult{name: "app", err: safeStart(func() error { return application.Start(runCtx) })}
	}()

	var runErr error
	var waitReason error
	pending := total

	var waitTimer *time.Timer
	var waitCh <-chan time.Time
	startWaitTimer := func(reason error) {
		if waitReason == nil && reason != nil {
			waitReason = reason
		}
		if waitCh != nil {
			return
		}
		waitTimer = time.NewTimer(waitTimeout)
		waitCh = waitTimer.C
	}
	defer func() {
		if waitTimer != nil {
			waitTimer.Stop()
		}
	}()

	for pending > 0 {
		select {
		case res := <-resultCh:
			pending--
			if res.err == nil {
				continue
			}

			wrapped := fmt.Errorf("%s failed: %w", res.name, res.err)
			if runErr == nil {
				runErr = wrapped
				cancel()
				startWaitTimer(wrapped)
				continue
			}
			runErr = errors.Join(runErr, wrapped)
		case <-ctx.Done():
			cancel()
			startWaitTimer(ctx.Err())
		case <-waitCh:
			timeoutErr := fmt.Errorf("shutdown wait timeout after %s with %d component(s) still running", waitTimeout, pending)
			if runErr != nil {
				return errors.Join(runErr, timeoutErr)
			}
			if waitReason != nil {
				return errors.Join(waitReason, timeoutErr)
			}
			return timeoutErr
		}
	}

	return runErr
}

func safeStart(startFn func() error) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic recovered: %v", r)
		}
	}()
	return startFn()
}

func normalizeComponentName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return "component"
	}
	return name
}

func resolveShutdownWaitTimeout(application *app.App) time.Duration {
	const defaultWaitTimeout = 10 * time.Second
	if application == nil || application.Config() == nil {
		return defaultWaitTimeout
	}

	timeout := application.Config().App.ShutdownTimeout
	if timeout <= 0 {
		return defaultWaitTimeout
	}
	return timeout
}

func isExpectedServeStop(err error) bool {
	if err == nil {
		return true
	}
	if errors.Is(err, http.ErrServerClosed) {
		return true
	}
	if errors.Is(err, net.ErrClosed) {
		return true
	}

	msg := strings.ToLower(strings.TrimSpace(err.Error()))
	if strings.Contains(msg, "server has been stopped") {
		return true
	}
	if strings.Contains(msg, "use of closed network connection") {
		return true
	}
	return false
}
