package gateway

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/yogayulanda/go-core/app"
	"github.com/yogayulanda/go-core/config"
	"github.com/yogayulanda/go-core/observability"
	"github.com/yogayulanda/go-core/response"
	grpcserver "github.com/yogayulanda/go-core/server/grpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/emptypb"
)

type slowEndpointServer struct{}

func TestGatewayConfiguredWriteTimeoutAllowsTwentySecondGRPCCall(t *testing.T) {
	grpcPort := availablePort(t)
	httpPort := availablePort(t)
	application, err := app.New(context.Background(), &config.Config{
		App:       config.AppConfig{ServiceName: "gateway-slow-response-test", Environment: "dev", LogLevel: "error", ShutdownTimeout: 25 * time.Second},
		GRPC:      config.GRPCConfig{Port: grpcPort},
		HTTP:      config.HTTPConfig{Port: httpPort, ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second, WriteTimeout: 60 * time.Second, IdleTimeout: 120 * time.Second, MaxBodyBytes: 1 << 20, TrustedProxyHops: 1},
		Databases: map[string]config.DBConfig{},
	})
	if err != nil {
		t.Fatalf("initialize app: %v", err)
	}
	grpcServer, err := grpcserver.New(application)
	if err != nil {
		t.Fatalf("initialize gRPC server: %v", err)
	}
	identity := make(chan [2]string, 1)
	grpcServer.Register(func(server *grpc.Server) {
		server.RegisterService(&grpc.ServiceDesc{
			ServiceName: "test.Slow",
			HandlerType: (*interface{})(nil),
			Methods: []grpc.MethodDesc{{
				MethodName: "Call",
				Handler: func(_ interface{}, ctx context.Context, _ func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
					handler := func(ctx context.Context, _ interface{}) (interface{}, error) {
						identity <- [2]string{observability.GetClientIP(ctx), observability.GetUserAgent(ctx)}
						if err := response.SetCode(ctx, "CORE-0000"); err != nil {
							return nil, err
						}
						select {
						case <-time.After(20 * time.Second):
							return &emptypb.Empty{}, nil
						case <-ctx.Done():
							return nil, ctx.Err()
						}
					}
					if interceptor == nil {
						return handler(ctx, nil)
					}
					return interceptor(ctx, nil, &grpc.UnaryServerInfo{FullMethod: "/test.Slow/Call"}, handler)
				},
			}},
		}, slowEndpointServer{})
	})
	gatewayServer, err := New(application, func(_ context.Context, mux *runtime.ServeMux) error {
		conn, err := grpc.NewClient(fmt.Sprintf("127.0.0.1:%d", grpcPort), grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			return err
		}
		application.Lifecycle().Register(func(context.Context) error { return conn.Close() })
		return mux.HandlePath(http.MethodGet, "/slow", func(w http.ResponseWriter, r *http.Request, _ map[string]string) {
			outgoing := metadata.NewOutgoingContext(r.Context(), metadata.Pairs(
				"x-request-id", r.Header.Get(requestIDHeader),
				"x-core-client-ip", r.Header.Get(clientIPHeader),
				"x-core-user-agent", r.Header.Get(userAgentHeader),
			))
			if err := conn.Invoke(outgoing, "/test.Slow/Call", &emptypb.Empty{}, &emptypb.Empty{}); err != nil {
				http.Error(w, "gRPC call failed", http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"ok":true}`))
		})
	})
	if err != nil {
		t.Fatalf("initialize gateway: %v", err)
	}
	go func() { _ = grpcServer.Start() }()
	go func() { _ = gatewayServer.Start() }()
	waitForListener(t, httpPort)
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = application.Lifecycle().Shutdown(ctx)
	}()

	client := &http.Client{Timeout: 25 * time.Second}
	started := time.Now()
	request, err := http.NewRequest(http.MethodGet, fmt.Sprintf("http://127.0.0.1:%d/slow", httpPort), nil)
	if err != nil {
		t.Fatalf("create slow request: %v", err)
	}
	request.Header.Set("X-Request-ID", "slow-req-1")
	request.Header.Set("X-Forwarded-For", "1.1.1.1, 10.0.0.2")
	request.Header.Set("User-Agent", "test-client/1.0")
	response, err := client.Do(request)
	if err != nil {
		t.Fatalf("slow gateway request failed: %v", err)
	}
	defer response.Body.Close()
	var body gatewayEnvelope
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decode response envelope: %v", err)
	}
	if response.StatusCode != http.StatusOK || body.Status != "SUCCESS" || body.ResponseCode != "CORE-0000" || body.RequestID != "slow-req-1" || response.Header.Get("X-Request-ID") != "slow-req-1" || response.Header.Get("X-Response-Code") != "" || response.Header.Get("Grpc-Metadata-X-Response-Code") != "" || time.Since(started) < 19*time.Second {
		t.Fatalf("slow request did not complete with full envelope: status=%d body=%+v duration=%s", response.StatusCode, body, time.Since(started))
	}
	if got := <-identity; got != [2]string{"1.1.1.1", "test-client/1.0"} {
		t.Fatalf("gateway identity metadata not restored at gRPC: %q", got)
	}
}

func waitForListener(t *testing.T, port int) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	address := fmt.Sprintf("127.0.0.1:%d", port)
	for time.Now().Before(deadline) {
		connection, err := net.DialTimeout("tcp", address, 20*time.Millisecond)
		if err == nil {
			_ = connection.Close()
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("listener %s did not become ready", address)
}

func availablePort(t *testing.T) int {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("reserve port: %v", err)
	}
	address, ok := listener.Addr().(*net.TCPAddr)
	if !ok {
		_ = listener.Close()
		t.Fatal("reserved listener did not return a TCP address")
	}
	if err := listener.Close(); err != nil {
		t.Fatalf("release reserved port: %v", err)
	}
	return address.Port
}
