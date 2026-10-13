package app

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/yogayulanda/go-core/logger"
	"github.com/yogayulanda/go-core/observability"
)

type inFlightRequest struct {
	requestID string
	gtid      string
	method    string
	startedAt time.Time
}

type runtimeState struct {
	draining atomic.Bool

	mu                   sync.Mutex
	drainHooks           []func()
	serverShutdowns      map[string]func(context.Context) error
	serverHookRegistered bool
	nextRequestID        uint64
	inFlight             map[uint64]inFlightRequest
	reportedRequestIDs   map[string]struct{}
}

func newRuntimeState() *runtimeState {
	return &runtimeState{
		serverShutdowns:    make(map[string]func(context.Context) error),
		inFlight:           make(map[uint64]inFlightRequest),
		reportedRequestIDs: make(map[string]struct{}),
	}
}

func (a *App) IsDraining() bool {
	return a.runtime != nil && a.runtime.draining.Load()
}

func (a *App) RegisterDrainHook(hook func()) {
	if a == nil || a.runtime == nil || hook == nil {
		return
	}
	a.runtime.mu.Lock()
	if a.runtime.draining.Load() {
		a.runtime.mu.Unlock()
		hook()
		return
	}
	a.runtime.drainHooks = append(a.runtime.drainHooks, hook)
	a.runtime.mu.Unlock()
}

func (a *App) BeginDrain() {
	if a == nil || a.runtime == nil || !a.runtime.draining.CompareAndSwap(false, true) {
		return
	}
	a.runtime.mu.Lock()
	hooks := append([]func(){}, a.runtime.drainHooks...)
	a.runtime.mu.Unlock()
	a.logger.LogService(context.Background(), logger.ServiceLog{Operation: "runtime_drain", Status: "started"})
	for _, hook := range hooks {
		hook()
	}
}

func (a *App) RegisterServerShutdown(name string, shutdown func(context.Context) error) {
	if a == nil || a.runtime == nil || shutdown == nil {
		return
	}
	a.runtime.mu.Lock()
	a.runtime.serverShutdowns[name] = shutdown
	register := !a.runtime.serverHookRegistered
	if register {
		a.runtime.serverHookRegistered = true
	}
	a.runtime.mu.Unlock()
	if register {
		a.lifecycle.Register(a.shutdownServers)
	}
}

func (a *App) shutdownServers(ctx context.Context) error {
	a.runtime.mu.Lock()
	names := make([]string, 0, len(a.runtime.serverShutdowns))
	for name := range a.runtime.serverShutdowns {
		names = append(names, name)
	}
	sort.Strings(names)
	shutdowns := make([]func(context.Context) error, 0, len(names))
	for _, name := range names {
		shutdowns = append(shutdowns, a.runtime.serverShutdowns[name])
	}
	a.runtime.mu.Unlock()

	type result struct{ err error }
	results := make(chan result, len(shutdowns))
	for _, shutdown := range shutdowns {
		stop := shutdown
		go func() { results <- result{err: stop(ctx)} }()
	}
	var joined error
	for range shutdowns {
		select {
		case outcome := <-results:
			if outcome.err != nil {
				joined = errors.Join(joined, outcome.err)
			}
		case <-ctx.Done():
			a.LogInflightCut()
			return errors.Join(joined, ctx.Err())
		}
	}
	return joined
}

func (a *App) TrackInFlight(ctx context.Context, method string) func() {
	if a == nil || a.runtime == nil {
		return func() {}
	}
	requestID := observability.GetRequestID(ctx)
	if requestID == "" {
		requestID = observability.GenerateRequestID()
	}
	a.runtime.mu.Lock()
	a.runtime.nextRequestID++
	id := a.runtime.nextRequestID
	a.runtime.inFlight[id] = inFlightRequest{requestID: requestID, gtid: observability.GetGTID(ctx), method: method, startedAt: time.Now()}
	a.runtime.mu.Unlock()
	return func() {
		a.runtime.mu.Lock()
		delete(a.runtime.inFlight, id)
		a.runtime.mu.Unlock()
	}
}

func (a *App) LogInflightCut() {
	if a == nil || a.runtime == nil {
		return
	}
	a.runtime.mu.Lock()
	requestsByID := make(map[string]inFlightRequest)
	for _, request := range a.runtime.inFlight {
		if _, exists := a.runtime.reportedRequestIDs[request.requestID]; exists {
			continue
		}
		if existing, ok := requestsByID[request.requestID]; ok {
			if existing.gtid == "" && request.gtid != "" {
				existing.gtid = request.gtid
			}
			if len(request.method) > len(existing.method) {
				existing.method = request.method
			}
			requestsByID[request.requestID] = existing
			continue
		}
		requestsByID[request.requestID] = request
	}
	requests := make([]inFlightRequest, 0, len(requestsByID))
	for requestID, request := range requestsByID {
		a.runtime.reportedRequestIDs[requestID] = struct{}{}
		requests = append(requests, request)
	}
	a.runtime.mu.Unlock()
	for _, request := range requests {
		ctx := observability.WithRequestID(context.Background(), request.requestID)
		if request.gtid != "" {
			ctx = observability.WithGTID(ctx, request.gtid)
		}
		fields := []logger.Field{
			{Key: "labels.request_id", Value: request.requestID},
			{Key: "app.log_type", Value: "access"},
			{Key: "app.method", Value: request.method},
			{Key: "app.inflight_duration_ms", Value: time.Since(request.startedAt).Milliseconds()},
		}
		if request.gtid != "" {
			fields = append(fields, logger.Field{Key: "labels.gtid", Value: request.gtid})
		}
		a.logger.Warn(ctx, "inflight_cut_by_shutdown", fields...)
	}
}

func (a *App) LogShutdownPhase(phase string, startedAt time.Time, err error) {
	if a == nil || a.logger == nil {
		return
	}
	status := "success"
	if err != nil {
		status = "failed"
	}
	a.logger.LogService(context.Background(), logger.ServiceLog{
		Operation:  "shutdown_phase",
		Status:     status,
		DurationMs: time.Since(startedAt).Milliseconds(),
		ErrorCode:  phase,
		Metadata: map[string]interface{}{
			"phase": phase,
			"error": errorString(err),
		},
	})
}

func errorString(err error) string {
	if err == nil {
		return ""
	}
	return fmt.Sprint(err)
}
