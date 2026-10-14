package gateway

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"

	"github.com/yogayulanda/go-core/app"
	"github.com/yogayulanda/go-core/config"
)

func BenchmarkGatewayHTTPEndToEnd(b *testing.B) {
	application := newGatewayBenchmarkApp(b)
	gatewayServer, err := New(application, func(_ context.Context, mux *runtime.ServeMux) error {
		return mux.HandlePath(http.MethodGet, "/benchmark", func(w http.ResponseWriter, _ *http.Request, _ map[string]string) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"ok":true}`))
		})
	})
	if err != nil {
		b.Fatal(err)
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		request := httptest.NewRequest(http.MethodGet, "/benchmark", nil)
		request.Header.Set("X-Request-ID", "benchmark-request-1")
		request.Header.Set("X-User-ID", "benchmark-user-1")
		recorder := httptest.NewRecorder()
		gatewayServer.server.Handler.ServeHTTP(recorder, request)
		if recorder.Code != http.StatusOK {
			b.Fatalf("gateway returned status %d", recorder.Code)
		}
	}
}

func BenchmarkGatewayRawHTTPEndToEnd(b *testing.B) {
	application := newGatewayBenchmarkApp(b)
	gatewayServer, err := New(application, func(_ context.Context, mux *runtime.ServeMux) error {
		return mux.HandlePath(http.MethodGet, "/benchmark-raw", Raw(func(w http.ResponseWriter, _ *http.Request, _ map[string]string) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"ok":true}`))
		}))
	})
	if err != nil {
		b.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodGet, "/benchmark-raw", nil)
	request.Header.Set("X-Request-ID", "benchmark-raw-request-1")

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		recorder := httptest.NewRecorder()
		gatewayServer.server.Handler.ServeHTTP(recorder, request)
		if recorder.Code != http.StatusOK {
			b.Fatalf("raw gateway returned status %d", recorder.Code)
		}
	}
}

func newGatewayBenchmarkApp(b *testing.B) *app.App {
	b.Helper()
	sink, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	if err != nil {
		b.Fatal(err)
	}
	previousStdout := os.Stdout
	os.Stdout = sink
	application, appErr := app.New(context.Background(), &config.Config{
		App: config.AppConfig{
			ServiceName:     "gateway-hot-path-benchmark",
			Environment:     "test",
			LogLevel:        "info",
			ShutdownTimeout: time.Second,
		},
		Databases: map[string]config.DBConfig{},
		Metrics:   config.MetricsConfig{Enabled: true},
		HTTP: config.HTTPConfig{
			MaxBodyBytes:     1 << 20,
			TrustedProxyHops: 1,
		},
	})
	os.Stdout = previousStdout
	if appErr != nil {
		_ = sink.Close()
		b.Fatal(appErr)
	}
	b.Cleanup(func() { _ = sink.Close() })
	return application
}
