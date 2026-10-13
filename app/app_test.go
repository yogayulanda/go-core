package app

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/yogayulanda/go-core/config"
	"github.com/yogayulanda/go-core/logger"
	"github.com/yogayulanda/go-core/messaging"
)

func TestNew_DBlessConfig(t *testing.T) {
	cfg := &config.Config{
		App: config.AppConfig{
			ServiceName:     "db-less-svc",
			Environment:     "test",
			LogLevel:        "info",
			ShutdownTimeout: 2 * time.Second,
		},
		Databases: map[string]config.DBConfig{},
	}

	core, err := New(context.Background(), cfg)
	if err != nil {
		t.Fatalf("expected app init success without db, got error: %v", err)
	}
	if core == nil {
		t.Fatalf("expected app instance")
	}
	if len(core.SQLAll()) != 0 {
		t.Fatalf("expected no sql connections for db-less config")
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := core.Start(ctx); err != nil {
		t.Fatalf("expected graceful start/shutdown without db, got error: %v", err)
	}
}

func TestNew_NilConfig_ReturnError(t *testing.T) {
	_, err := New(context.Background(), nil)
	if err == nil {
		t.Fatalf("expected error for nil config")
	}
}

type warningRecorder struct {
	warnings []string
	fields   [][]logger.Field
}

func (r *warningRecorder) Warn(_ context.Context, message string, fields ...logger.Field) {
	r.warnings = append(r.warnings, message)
	r.fields = append(r.fields, fields)
}

func (*warningRecorder) Info(context.Context, string, ...logger.Field)         {}
func (*warningRecorder) Error(context.Context, string, ...logger.Field)        {}
func (*warningRecorder) Debug(context.Context, string, ...logger.Field)        {}
func (*warningRecorder) LogService(context.Context, logger.ServiceLog)         {}
func (*warningRecorder) LogDB(context.Context, logger.DBLog)                   {}
func (*warningRecorder) LogEvent(context.Context, logger.EventLog)             {}
func (*warningRecorder) LogTransaction(context.Context, logger.TransactionLog) {}
func (r *warningRecorder) WithComponent(string) logger.Logger                  { return r }

func TestLogIgnoredEnvironmentVariables_OneNamesOnlyWarning(t *testing.T) {
	recorder := &warningRecorder{}
	logIgnoredEnvironmentVariables(context.Background(), recorder, []string{"LOG_TIMEZONE", "REDIS_ENABLED"})

	if len(recorder.warnings) != 1 {
		t.Fatalf("expected one warning, got %d", len(recorder.warnings))
	}
	if recorder.warnings[0] != "ignored removed environment variables" {
		t.Fatalf("unexpected warning message: %q", recorder.warnings[0])
	}
	if len(recorder.fields[0]) != 1 || recorder.fields[0][0].Key != "environment_variables" {
		t.Fatalf("expected only the removed variable names field, got %#v", recorder.fields[0])
	}
}

func TestKafkaHelpers_Disabled_ReturnErrKafkaDisabled(t *testing.T) {
	cfg := &config.Config{
		App: config.AppConfig{
			ServiceName:     "kafka-disabled-test",
			Environment:     "test",
			LogLevel:        "info",
			ShutdownTimeout: 2 * time.Second,
		},
		Databases: map[string]config.DBConfig{},
	}

	application, err := New(context.Background(), cfg)
	if err != nil {
		t.Fatalf("unexpected app init error: %v", err)
	}

	_, err = application.NewKafkaPublisher()
	if !errors.Is(err, ErrKafkaDisabled) {
		t.Fatalf("expected ErrKafkaDisabled from publisher, got: %v", err)
	}

	_, err = application.NewKafkaConsumer(
		"topic",
		"group",
		func(ctx context.Context, msg messaging.Message) error { return nil },
	)
	if !errors.Is(err, ErrKafkaDisabled) {
		t.Fatalf("expected ErrKafkaDisabled from consumer, got: %v", err)
	}
}
