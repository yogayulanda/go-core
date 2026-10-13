package app

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/yogayulanda/go-core/config"
	"github.com/yogayulanda/go-core/logger"
	"github.com/yogayulanda/go-core/observability"
)

func TestBeginDrainRunsHooksOnce(t *testing.T) {
	application, err := New(context.Background(), &config.Config{App: config.AppConfig{ServiceName: "drain-test", Environment: "dev", LogLevel: "error", ShutdownTimeout: time.Second}, Databases: map[string]config.DBConfig{}})
	if err != nil {
		t.Fatalf("initialize app: %v", err)
	}
	var calls atomic.Int32
	application.RegisterDrainHook(func() { calls.Add(1) })
	application.BeginDrain()
	application.BeginDrain()
	if !application.IsDraining() || calls.Load() != 1 {
		t.Fatalf("drain state/hooks incorrect: draining=%v calls=%d", application.IsDraining(), calls.Load())
	}
}

func TestServerShutdownHooksRunInParallel(t *testing.T) {
	application, err := New(context.Background(), &config.Config{App: config.AppConfig{ServiceName: "shutdown-group-test", Environment: "dev", LogLevel: "error", ShutdownTimeout: time.Second}, Databases: map[string]config.DBConfig{}})
	if err != nil {
		t.Fatalf("initialize app: %v", err)
	}
	var started atomic.Int32
	barrier := make(chan struct{})
	for _, name := range []string{"http_gateway", "grpc_server"} {
		application.RegisterServerShutdown(name, func(ctx context.Context) error {
			if started.Add(1) == 2 {
				close(barrier)
			}
			select {
			case <-barrier:
				return nil
			case <-ctx.Done():
				return ctx.Err()
			}
		})
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := application.Lifecycle().Shutdown(ctx); err != nil {
		t.Fatalf("parallel server shutdown: %v", err)
	}
	if started.Load() != 2 {
		t.Fatalf("expected both shutdown hooks to start in parallel, got %d", started.Load())
	}
}

type inflightRecorder struct {
	warnings []string
	fields   [][]logger.Field
}

func (r *inflightRecorder) Info(context.Context, string, ...logger.Field)  {}
func (r *inflightRecorder) Error(context.Context, string, ...logger.Field) {}
func (r *inflightRecorder) Debug(context.Context, string, ...logger.Field) {}
func (r *inflightRecorder) Warn(_ context.Context, message string, fields ...logger.Field) {
	r.warnings = append(r.warnings, message)
	r.fields = append(r.fields, fields)
}
func (*inflightRecorder) LogService(context.Context, logger.ServiceLog)         {}
func (*inflightRecorder) LogDB(context.Context, logger.DBLog)                   {}
func (*inflightRecorder) LogEvent(context.Context, logger.EventLog)             {}
func (*inflightRecorder) LogTransaction(context.Context, logger.TransactionLog) {}
func (r *inflightRecorder) WithComponent(string) logger.Logger                  { return r }

func TestLogInflightCutDeduplicatesGatewayAndGRPCByRequestID(t *testing.T) {
	application, err := New(context.Background(), &config.Config{App: config.AppConfig{ServiceName: "inflight-test", Environment: "dev", LogLevel: "error", ShutdownTimeout: time.Second}, Databases: map[string]config.DBConfig{}})
	if err != nil {
		t.Fatalf("initialize app: %v", err)
	}
	recorder := &inflightRecorder{}
	application.logger = recorder
	ctx := observability.WithRequestID(observability.EnsureFieldBag(context.Background()), "req-inflight")
	_ = observability.SetLabel(ctx, "gtid", "gtid-1")
	finishGateway := application.TrackInFlight(ctx, "POST /v1/payments")
	finishGRPC := application.TrackInFlight(ctx, "/payments.Service/Pay")
	application.LogInflightCut()
	finishGateway()
	finishGRPC()
	application.LogInflightCut()
	if len(recorder.warnings) != 1 || recorder.warnings[0] != "inflight_cut_by_shutdown" {
		t.Fatalf("expected one deduplicated cutoff warning, got %#v", recorder.warnings)
	}
	if len(recorder.fields[0]) < 2 || recorder.fields[0][0].Key != "labels.request_id" {
		t.Fatalf("cutoff warning missing request identity: %#v", recorder.fields[0])
	}
}
