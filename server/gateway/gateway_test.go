package gateway

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
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

func TestGatewayDoesNotEnableH2C(t *testing.T) {
	application := newTestApp(t, &config.Config{
		App:       config.AppConfig{ServiceName: "gateway-h2c-test", Environment: "dev", LogLevel: "error", ShutdownTimeout: time.Second},
		HTTP:      config.HTTPConfig{Port: 0},
		Databases: map[string]config.DBConfig{},
	})
	gateway, err := New(application, func(context.Context, *runtime.ServeMux) error { return nil })
	if err != nil {
		t.Fatalf("create gateway: %v", err)
	}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	serveDone := make(chan error, 1)
	go func() { serveDone <- gateway.server.Serve(listener) }()
	t.Cleanup(func() {
		_ = gateway.server.Close()
		<-serveDone
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		if err := application.Lifecycle().Shutdown(ctx); err != nil {
			t.Errorf("shutdown application: %v", err)
		}
	})

	conn, err := net.DialTimeout("tcp", listener.Addr().String(), time.Second)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(time.Second))
	_, err = fmt.Fprint(conn, "GET /health HTTP/1.1\r\nHost: localhost\r\nConnection: Upgrade, HTTP2-Settings\r\nUpgrade: h2c\r\nHTTP2-Settings: AAMAAABkAAQAAP__\r\n\r\n")
	if err != nil {
		t.Fatalf("send h2c upgrade request: %v", err)
	}
	response, err := http.ReadResponse(bufio.NewReader(conn), nil)
	if err != nil {
		t.Fatalf("read HTTP response: %v", err)
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusSwitchingProtocols || strings.EqualFold(response.Header.Get("Upgrade"), "h2c") {
		t.Fatalf("cleartext h2c upgrade was accepted: status=%d headers=%v", response.StatusCode, response.Header)
	}

	priorKnowledgeConn, err := net.DialTimeout("tcp", listener.Addr().String(), time.Second)
	if err != nil {
		t.Fatalf("connect for h2c prior knowledge: %v", err)
	}
	defer priorKnowledgeConn.Close()
	_ = priorKnowledgeConn.SetDeadline(time.Now().Add(time.Second))
	if _, err := io.WriteString(priorKnowledgeConn, "PRI * HTTP/2.0\r\n\r\nSM\r\n\r\n"); err != nil {
		t.Fatalf("send HTTP/2 client preface: %v", err)
	}
	if _, err := priorKnowledgeConn.Write([]byte{0, 0, 0, 4, 0, 0, 0, 0, 0}); err != nil {
		t.Fatalf("send HTTP/2 settings frame: %v", err)
	}
	responseFrame, err := bufio.NewReader(priorKnowledgeConn).Peek(9)
	if err == nil && responseFrame[3] == 4 && responseFrame[5] == 0 && responseFrame[6] == 0 && responseFrame[7] == 0 && responseFrame[8] == 0 {
		t.Fatalf("cleartext HTTP/2 prior-knowledge connection received a SETTINGS frame: %v", responseFrame)
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
