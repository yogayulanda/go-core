package app

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/yogayulanda/go-core/config"
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
