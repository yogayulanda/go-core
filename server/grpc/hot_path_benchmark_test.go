package grpc

import (
	"context"
	"os"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	"github.com/yogayulanda/go-core/app"
	"github.com/yogayulanda/go-core/config"
	"github.com/yogayulanda/go-core/observability"
)

func BenchmarkUnaryInterceptorChain(b *testing.B) {
	application := newUnaryBenchmarkApp(b)
	if err := observability.RegisterPropagatedLabel("benchmark"); err != nil {
		b.Fatal(err)
	}

	interceptors := []grpc.UnaryServerInterceptor{
		requestIDInterceptor(),
		inFlightInterceptor(application),
		loggingInterceptor(application),
		authInterceptorWithLogger(nil, application.Logger()),
		metricsInterceptor(application),
	}
	info := &grpc.UnaryServerInfo{FullMethod: "/benchmark.v1.Service/Unary"}
	call := chainUnaryBenchmark(interceptors, info, func(context.Context, interface{}) (interface{}, error) {
		return "ok", nil
	})
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(
		"x-request-id", "benchmark-request-1",
		"x-user-id", "benchmark-user-1",
		"x-label-benchmark", "enabled",
	))
	request := struct{ Value string }{Value: "payload"}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := call(ctx, request); err != nil {
			b.Fatal(err)
		}
	}
}

func newUnaryBenchmarkApp(b *testing.B) *app.App {
	b.Helper()
	sink, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	if err != nil {
		b.Fatal(err)
	}
	previousStdout := os.Stdout
	os.Stdout = sink
	application, appErr := app.New(context.Background(), &config.Config{
		App: config.AppConfig{
			ServiceName:     "grpc-hot-path-benchmark",
			Environment:     "test",
			LogLevel:        "info",
			ShutdownTimeout: time.Second,
		},
		Databases: map[string]config.DBConfig{},
		Metrics:   config.MetricsConfig{Enabled: true},
	})
	os.Stdout = previousStdout
	if appErr != nil {
		_ = sink.Close()
		b.Fatal(appErr)
	}
	b.Cleanup(func() { _ = sink.Close() })
	return application
}

func chainUnaryBenchmark(
	interceptors []grpc.UnaryServerInterceptor,
	info *grpc.UnaryServerInfo,
	final grpc.UnaryHandler,
) grpc.UnaryHandler {
	handler := final
	for index := len(interceptors) - 1; index >= 0; index-- {
		interceptor := interceptors[index]
		next := handler
		handler = func(ctx context.Context, request interface{}) (interface{}, error) {
			return interceptor(ctx, request, info, next)
		}
	}
	return handler
}
