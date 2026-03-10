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
