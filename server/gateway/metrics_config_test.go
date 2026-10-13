package gateway

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/yogayulanda/go-core/app"
	"github.com/yogayulanda/go-core/config"
)

func TestGatewayMetricsEndpointRequiresExplicitEnablement(t *testing.T) {
	for _, enabled := range []bool{false, true} {
		t.Run(map[bool]string{false: "default_disabled", true: "enabled"}[enabled], func(t *testing.T) {
			application, err := app.New(context.Background(), &config.Config{
				App:       config.AppConfig{ServiceName: "metrics-config-test", Environment: "dev", LogLevel: "error", ShutdownTimeout: time.Second},
				Metrics:   config.MetricsConfig{Enabled: enabled},
				Databases: map[string]config.DBConfig{},
			})
			if err != nil {
				t.Fatal(err)
			}
			gatewayServer, err := New(application, func(context.Context, *runtime.ServeMux) error { return nil })
			if err != nil {
				t.Fatal(err)
			}
			recorder := httptest.NewRecorder()
			gatewayServer.server.Handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/metrics", nil))
			wantStatus := http.StatusNotFound
			if enabled {
				wantStatus = http.StatusOK
			}
			if recorder.Code != wantStatus {
				t.Fatalf("/metrics status = %d, want %d", recorder.Code, wantStatus)
			}
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			if err := application.Lifecycle().Shutdown(ctx); err != nil {
				t.Fatal(err)
			}
		})
	}
}
