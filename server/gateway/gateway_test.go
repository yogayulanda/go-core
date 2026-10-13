package gateway

import (
	"context"
	"testing"
	"time"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/yogayulanda/go-core/config"
)

func TestNew_NilApplication_ReturnError(t *testing.T) {
	_, err := New(nil, func(ctx context.Context, mux *runtime.ServeMux) error { return nil })
	if err == nil {
		t.Fatalf("expected error for nil application")
	}
}

func TestNew_NilRegisterFunc_ReturnError(t *testing.T) {
	application := newTestApp(t, &config.Config{
		App: config.AppConfig{
			ServiceName:     "gateway-new-test",
			Environment:     "test",
			LogLevel:        "error",
			ShutdownTimeout: time.Second,
		},
		Databases: map[string]config.DBConfig{},
	})

	_, err := New(application, nil)
	if err == nil {
		t.Fatalf("expected error for nil registerFunc")
	}
}

func TestNew_UsesConfiguredHTTPTimeouts(t *testing.T) {
	application := newTestApp(t, &config.Config{
		App:       config.AppConfig{ServiceName: "gateway-timeout-test", Environment: "dev", LogLevel: "error", ShutdownTimeout: time.Second},
		HTTP:      config.HTTPConfig{Port: 0, ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second, WriteTimeout: 60 * time.Second, IdleTimeout: 120 * time.Second, MaxBodyBytes: 1 << 20},
		Databases: map[string]config.DBConfig{},
	})
	gateway, err := New(application, func(context.Context, *runtime.ServeMux) error { return nil })
	if err != nil {
		t.Fatalf("create gateway: %v", err)
	}
	if gateway.server.ReadHeaderTimeout != 5*time.Second || gateway.server.ReadTimeout != 15*time.Second || gateway.server.WriteTimeout != 60*time.Second || gateway.server.IdleTimeout != 120*time.Second {
		t.Fatalf("gateway did not apply configured timeouts: %+v", gateway.server)
	}
}

func TestNew_RejectsPprofInProduction(t *testing.T) {
	application := newTestApp(t, &config.Config{
		App:       config.AppConfig{ServiceName: "gateway-pprof-prod-test", Environment: "prod", LogLevel: "error", ShutdownTimeout: time.Second},
		HTTP:      config.HTTPConfig{PprofEnabled: true},
		Databases: map[string]config.DBConfig{},
	})
	if _, err := New(application, func(context.Context, *runtime.ServeMux) error { return nil }); err == nil {
		t.Fatal("expected production pprof refusal")
	}
}
