package server

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/yogayulanda/go-core/logger"
	grpc_health_v1 "google.golang.org/grpc/health/grpc_health_v1"
)

func TestDescribeFromProto_HealthService(t *testing.T) {
	httpEndpoints, grpcMethods := DescribeFromProto(grpc_health_v1.File_grpc_health_v1_health_proto, true)

	if len(grpcMethods) == 0 {
		t.Fatalf("expected grpc methods")
	}
	hasCheck := false
	for _, m := range grpcMethods {
		if strings.Contains(m, "/grpc.health.v1/Health/Check") {
			hasCheck = true
			break
		}
	}
	if !hasCheck {
		t.Fatalf("expected health check method in grpc methods")
	}

	hasCoreHealth := false
	for _, ep := range httpEndpoints {
		if ep == "GET /health" {
			hasCoreHealth = true
			break
		}
	}
	if !hasCoreHealth {
		t.Fatalf("expected core /health endpoint")
	}
}

func TestWaitForTCP_Ready(t *testing.T) {
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		skipIfListenNotPermitted(t, err)
		t.Fatalf("listen failed: %v", err)
	}
	defer lis.Close()

	port := portFromAddr(t, lis.Addr())
	if !waitForTCP(context.Background(), port, time.Second) {
		t.Fatalf("expected tcp ready")
	}
}

func TestWaitForHTTPHealth_Ready(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		skipIfListenNotPermitted(t, err)
		t.Fatalf("listen failed: %v", err)
	}
	defer lis.Close()

	srv := &http.Server{
		Handler:           mux,
		ReadHeaderTimeout: time.Second,
	}
	go func() { _ = srv.Serve(lis) }()
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_ = srv.Shutdown(ctx)
	}()

	port := portFromAddr(t, lis.Addr())
	if !waitForHTTPHealth(context.Background(), port, time.Second, false) {
		t.Fatalf("expected http health ready")
	}
}

func TestWaitForHTTPHealth_TLSReady(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	srv := newTLSServerOrSkip(t, mux)
	defer srv.Close()

	u, err := url.Parse(srv.URL)
	if err != nil {
		t.Fatalf("parse url failed: %v", err)
	}
	_, portRaw, err := net.SplitHostPort(u.Host)
	if err != nil {
		t.Fatalf("split host port failed: %v", err)
	}
	port, err := strconv.Atoi(portRaw)
	if err != nil {
		t.Fatalf("parse port failed: %v", err)
	}

	if !waitForHTTPHealth(context.Background(), port, time.Second, true) {
		t.Fatalf("expected https health ready")
	}
}

func TestNormalizePath(t *testing.T) {
	if got := normalizePath("health"); got != "/health" {
		t.Fatalf("unexpected path: %s", got)
	}
	if got := normalizePath("/ready"); got != "/ready" {
		t.Fatalf("unexpected path: %s", got)
	}
}

func TestWaitForHTTPHealth_Timeout(t *testing.T) {
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		skipIfListenNotPermitted(t, err)
		t.Fatalf("listen failed: %v", err)
	}
	port := portFromAddr(t, lis.Addr())
	_ = lis.Close()

	if waitForHTTPHealth(context.Background(), port, 100*time.Millisecond, false) {
		t.Fatalf("expected not ready")
	}
}

func TestLogStartupReadiness_AllReady(t *testing.T) {
	grpcLis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		skipIfListenNotPermitted(t, err)
		t.Fatalf("grpc listen failed: %v", err)
	}
	defer grpcLis.Close()

	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	httpLis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		skipIfListenNotPermitted(t, err)
		t.Fatalf("http listen failed: %v", err)
	}
	defer httpLis.Close()

	httpServer := &http.Server{
		Handler:           mux,
		ReadHeaderTimeout: time.Second,
	}
	go func() { _ = httpServer.Serve(httpLis) }()
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_ = httpServer.Shutdown(ctx)
	}()

	log := &fakeLogger{}
	LogStartupReadiness(
		context.Background(),
		log,
		portFromAddr(t, grpcLis.Addr()),
		portFromAddr(t, httpLis.Addr()),
		time.Second,
		false,
	)

	if !log.containsService("grpc_readiness", "success") {
		t.Fatalf("expected service log: grpc_readiness success")
	}
	if !log.containsService("http_gateway_readiness", "success") {
		t.Fatalf("expected service log: http_gateway_readiness success")
	}
	if !log.containsService("service_readiness", "success") {
		t.Fatalf("expected service log: service_readiness success")
	}
	if log.countWarn() != 0 {
		t.Fatalf("expected no warn logs, got %d", log.countWarn())
	}
}

func TestLogStartupReadiness_Timeout(t *testing.T) {
	grpcPort := getFreePort(t)
	httpPort := getFreePort(t)

	log := &fakeLogger{}
	LogStartupReadiness(
		context.Background(),
		log,
		grpcPort,
		httpPort,
		150*time.Millisecond,
		false,
	)

	if !log.containsService("grpc_readiness", "failed") {
		t.Fatalf("expected service log: grpc_readiness failed")
	}
	if !log.containsService("http_gateway_readiness", "failed") {
		t.Fatalf("expected service log: http_gateway_readiness failed")
	}
	if log.containsService("service_readiness", "success") {
		t.Fatalf("did not expect service_readiness success log on timeout")
	}
}

func getFreePort(t *testing.T) int {
	t.Helper()
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		skipIfListenNotPermitted(t, err)
		t.Fatalf("listen failed: %v", err)
	}
	port := portFromAddr(t, lis.Addr())
	_ = lis.Close()
	return port
}

func portFromAddr(t *testing.T, addr net.Addr) int {
	t.Helper()
	tcpAddr, ok := addr.(*net.TCPAddr)
	if !ok || tcpAddr == nil {
		t.Fatalf("expected *net.TCPAddr, got %T", addr)
	}
	return tcpAddr.Port
}

func skipIfListenNotPermitted(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		return
	}
	if errors.Is(err, net.ErrClosed) || strings.Contains(err.Error(), "operation not permitted") {
		t.Skipf("listen not permitted in current environment: %v", err)
	}
}

func newTLSServerOrSkip(t *testing.T, handler http.Handler) (srv *httptest.Server) {
	t.Helper()

	defer func() {
		if r := recover(); r != nil {
			if strings.Contains(strings.ToLower(toString(r)), "operation not permitted") {
				t.Skipf("tls test server listen not permitted in current environment: %v", r)
			}
			panic(r)
		}
	}()

	return httptest.NewTLSServer(handler)
}

func toString(v interface{}) string {
	switch x := v.(type) {
	case string:
		return x
	case error:
		return x.Error()
	default:
		return ""
	}
}

type fakeLogger struct {
	mu          sync.Mutex
	infoLogs    []string
	warnLogs    []string
	serviceLogs []logger.ServiceLog
}

func (f *fakeLogger) Info(ctx context.Context, msg string, fields ...logger.Field) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.infoLogs = append(f.infoLogs, msg)
}

func (f *fakeLogger) Error(ctx context.Context, msg string, fields ...logger.Field) {}
func (f *fakeLogger) Debug(ctx context.Context, msg string, fields ...logger.Field) {}

func (f *fakeLogger) Warn(ctx context.Context, msg string, fields ...logger.Field) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.warnLogs = append(f.warnLogs, msg)
}

func (f *fakeLogger) LogService(ctx context.Context, s logger.ServiceLog) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.serviceLogs = append(f.serviceLogs, s)
}

func (f *fakeLogger) LogDB(ctx context.Context, d logger.DBLog) {}

func (f *fakeLogger) LogEvent(ctx context.Context, e logger.EventLog) {}

func (f *fakeLogger) LogTransaction(ctx context.Context, tx logger.TransactionLog) {}

func (f *fakeLogger) WithComponent(component string) logger.Logger { return f }

func (f *fakeLogger) containsInfo(msg string) bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, got := range f.infoLogs {
		if got == msg {
			return true
		}
	}
	return false
}

func (f *fakeLogger) containsWarn(msg string) bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, got := range f.warnLogs {
		if got == msg {
			return true
		}
	}
	return false
}

func (f *fakeLogger) countWarn() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.warnLogs)
}

func (f *fakeLogger) containsService(operation string, status string) bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, got := range f.serviceLogs {
		if got.Operation == operation && got.Status == status {
			return true
		}
	}
	return false
}
