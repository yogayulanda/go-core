package grpc

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/yogayulanda/go-core/app"
	"github.com/yogayulanda/go-core/config"
	coreErrors "github.com/yogayulanda/go-core/errors"
	"github.com/yogayulanda/go-core/logger"
	"github.com/yogayulanda/go-core/observability"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

type accessLogRecorder struct {
	entries []logger.ServiceLog
}

func (*accessLogRecorder) Info(context.Context, string, ...logger.Field)  {}
func (*accessLogRecorder) Error(context.Context, string, ...logger.Field) {}
func (*accessLogRecorder) Debug(context.Context, string, ...logger.Field) {}
func (*accessLogRecorder) Warn(context.Context, string, ...logger.Field)  {}
func (r *accessLogRecorder) LogService(_ context.Context, entry logger.ServiceLog) {
	r.entries = append(r.entries, entry)
}
func (*accessLogRecorder) LogDB(context.Context, logger.DBLog)                   {}
func (*accessLogRecorder) LogEvent(context.Context, logger.EventLog)             {}
func (*accessLogRecorder) LogTransaction(context.Context, logger.TransactionLog) {}
func (r *accessLogRecorder) WithComponent(string) logger.Logger                  { return r }

func TestRequestIDInterceptorValidatesIDAndRestoresIncomingIdentity(t *testing.T) {
	if err := observability.RegisterPropagatedLabel("client_app"); err != nil {
		t.Fatalf("register label: %v", err)
	}
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(
		"x-request-id", "../../etc",
		"x-user-id", "user-7",
		"x-label-client_app", "sitepat",
		"x-label-unregistered", "ignored",
	))
	var handlerContext context.Context
	interceptor := requestIDInterceptor()
	_, err := interceptor(ctx, nil, &grpc.UnaryServerInfo{FullMethod: "/test.Service/Call"}, func(ctx context.Context, _ interface{}) (interface{}, error) {
		handlerContext = ctx
		return nil, nil
	})
	if err != nil {
		t.Fatalf("interceptor error: %v", err)
	}
	if requestID := observability.GetRequestID(handlerContext); observability.IsValidRequestID(requestID) == false || requestID == "../../etc" {
		t.Fatalf("invalid request ID was not replaced: %q", requestID)
	}
	if !observability.RequestIDReplaced(handlerContext) {
		t.Fatal("expected replacement marker")
	}
	if got := observability.GetUserID(handlerContext); got != "user-7" {
		t.Fatalf("unexpected restored user ID: %q", got)
	}
	if got := observability.Labels(handlerContext)["client_app"]; got != "sitepat" {
		t.Fatalf("unexpected restored label: %q", got)
	}
	if _, ok := observability.Labels(handlerContext)["unregistered"]; ok {
		t.Fatal("unregistered propagated label was restored")
	}
}

func TestLoggingInterceptor_UnregisteredCodeOneWarningAccessLog(t *testing.T) {
	recorder := &accessLogRecorder{}
	interceptor := loggingInterceptorWithLogger(recorder, false)
	_, err := interceptor(context.Background(), nil, &grpc.UnaryServerInfo{FullMethod: "/test.Service/Call"}, func(context.Context, interface{}) (interface{}, error) {
		return nil, coreErrors.New("CUSTOM-UNREGISTERED", "technical secret detail")
	})
	if err == nil {
		t.Fatal("expected gRPC error")
	}
	if len(recorder.entries) != 1 {
		t.Fatalf("expected exactly one access log, got %d", len(recorder.entries))
	}
	entry := recorder.entries[0]
	if entry.LogType != "access" || entry.Class != "ERROR" || entry.ResponseCode != "CORE-9000" || entry.UnregisteredCode != "CUSTOM-UNREGISTERED" {
		t.Fatalf("unexpected unknown-code access log: %+v", entry)
	}
	if entry.ErrorMessage != "" {
		t.Fatalf("unregistered technical detail should not be copied to the access log: %q", entry.ErrorMessage)
	}
	statusInfo, ok := coreErrors.InfoFromGRPC(err)
	if !ok || statusInfo.Code != "CORE-9000" {
		t.Fatalf("expected CORE-9000 response code, got %+v, %v", statusInfo, ok)
	}
}

func TestLoggingInterceptor_NonAppErrorSanitizesAndLogsOnce(t *testing.T) {
	recorder := &accessLogRecorder{}
	interceptor := loggingInterceptorWithLogger(recorder, false)
	_, err := interceptor(context.Background(), nil, &grpc.UnaryServerInfo{FullMethod: "/test.Service/Call"}, func(context.Context, interface{}) (interface{}, error) {
		return nil, errors.New("raw internal failure")
	})
	if err == nil || len(recorder.entries) != 1 {
		t.Fatalf("expected one mapped error and one access log, got err=%v logs=%d", err, len(recorder.entries))
	}
	if recorder.entries[0].ResponseCode != "CORE-9000" || recorder.entries[0].Class != "ERROR" {
		t.Fatalf("unexpected mapped access entry: %+v", recorder.entries[0])
	}
}

type directUnknownCodeService struct{}

func TestDirectGRPCServer_UnregisteredCodeWritesOneWarningAccessLog(t *testing.T) {
	previousStdout := os.Stdout
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("create stdout pipe: %v", err)
	}
	os.Stdout = writer
	defer func() {
		os.Stdout = previousStdout
		_ = reader.Close()
		_ = writer.Close()
	}()

	application, err := app.New(context.Background(), &config.Config{
		App:       config.AppConfig{ServiceName: "grpc-unregistered-test", Environment: "dev", LogLevel: "debug", ShutdownTimeout: time.Second},
		Databases: map[string]config.DBConfig{},
	})
	if err != nil {
		t.Fatalf("initialize app: %v", err)
	}
	server, err := New(application)
	if err != nil {
		t.Fatalf("initialize gRPC server: %v", err)
	}
	server.Register(func(server *grpc.Server) {
		server.RegisterService(&grpc.ServiceDesc{
			ServiceName: "test.UnknownCode",
			HandlerType: (*interface{})(nil),
			Methods: []grpc.MethodDesc{{
				MethodName: "Call",
				Handler: func(_ interface{}, ctx context.Context, _ func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
					handler := func(context.Context, interface{}) (interface{}, error) {
						return nil, coreErrors.New("CUSTOM-UNREGISTERED", "technical secret detail")
					}
					if interceptor == nil {
						return handler(ctx, nil)
					}
					return interceptor(ctx, nil, &grpc.UnaryServerInfo{Server: nil, FullMethod: "/test.UnknownCode/Call"}, handler)
				},
			}},
		}, directUnknownCodeService{})
	})
	serveDone := make(chan error, 1)
	go func() { serveDone <- server.Start() }()
	conn, err := grpc.NewClient(server.lis.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("dial gRPC server: %v", err)
	}
	callErr := conn.Invoke(metadata.NewOutgoingContext(context.Background(), metadata.Pairs("x-request-id", "direct-1")), "/test.UnknownCode/Call", &emptypb.Empty{}, &emptypb.Empty{})
	if callErr == nil {
		t.Fatal("expected unregistered-code RPC error")
	}
	_ = conn.Close()
	server.server.Stop()
	_ = server.lis.Close()
	<-serveDone
	_ = writer.Close()
	os.Stdout = previousStdout
	raw, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("read captured logs: %v", err)
	}

	accessCount := 0
	for _, line := range strings.Split(strings.TrimSpace(string(raw)), "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		var entry map[string]interface{}
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			continue
		}
		if entry["app.log_type"] != "access" {
			continue
		}
		accessCount++
		if entry["log.level"] != "warn" || entry["app.response_code"] != "CORE-9000" || entry["app.unregistered_code"] != "CUSTOM-UNREGISTERED" {
			t.Fatalf("unexpected access log: %#v", entry)
		}
		if _, leaked := entry["error.message"]; leaked {
			t.Fatalf("unregistered technical error message leaked: %#v", entry)
		}
	}
	if accessCount != 1 {
		t.Fatalf("expected exactly one gRPC access log, got %d; output=%s", accessCount, raw)
	}
}

func TestDirectGRPCServer_AuthRejectionLogsOnceWithoutAuthorizationOrToken(t *testing.T) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate verifier key: %v", err)
	}
	publicDER, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	if err != nil {
		t.Fatalf("marshal verifier key: %v", err)
	}
	publicPEM := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: publicDER})

	previousStdout := os.Stdout
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("create stdout pipe: %v", err)
	}
	os.Stdout = writer
	defer func() {
		os.Stdout = previousStdout
		_ = reader.Close()
		_ = writer.Close()
	}()

	application, err := app.New(context.Background(), &config.Config{
		App:       config.AppConfig{ServiceName: "grpc-auth-rejection-test", Environment: "dev", LogLevel: "debug", ShutdownTimeout: time.Second},
		GRPC:      config.GRPCConfig{Port: 0},
		Auth:      config.AuthConfig{InternalJWT: config.InternalJWTConfig{Enabled: true, PublicKey: string(publicPEM)}},
		Databases: map[string]config.DBConfig{},
	})
	if err != nil {
		t.Fatalf("initialize app: %v", err)
	}
	server, err := New(application)
	if err != nil {
		t.Fatalf("initialize gRPC server: %v", err)
	}
	server.Register(func(server *grpc.Server) {
		server.RegisterService(directServiceDescription("test.AuthRequired", func(context.Context) error { return nil }), directUnknownCodeService{})
	})
	serveDone := make(chan error, 1)
	go func() { serveDone <- server.Start() }()
	conn, err := grpc.NewClient(server.lis.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("dial gRPC server: %v", err)
	}
	const secretToken = "secret-token-never-log"
	callErr := conn.Invoke(metadata.NewOutgoingContext(context.Background(), metadata.Pairs("authorization", "Bearer "+secretToken, "x-request-id", "auth-reject-1")), "/test.AuthRequired/Call", &emptypb.Empty{}, &emptypb.Empty{})
	if status.Code(callErr) != codes.Unauthenticated {
		t.Fatalf("expected gRPC Unauthenticated (HTTP 401 mapping), got %v", callErr)
	}
	unauthorizedDef, registered := coreErrors.Lookup("CORE-1001")
	if !registered || unauthorizedDef.HTTPStatus != 401 || unauthorizedDef.GRPCCode != codes.Unauthenticated {
		t.Fatalf("expected CORE-1001 to map to HTTP 401/gRPC Unauthenticated, got %+v", unauthorizedDef)
	}
	_ = conn.Close()
	server.server.Stop()
	_ = server.lis.Close()
	<-serveDone
	_ = writer.Close()
	os.Stdout = previousStdout
	raw, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("read captured logs: %v", err)
	}
	output := string(raw)
	if strings.Contains(output, "Authorization") || strings.Contains(output, secretToken) {
		t.Fatalf("authentication header or token leaked to logs: %s", output)
	}

	accessCount := 0
	for _, line := range strings.Split(strings.TrimSpace(output), "\n") {
		var entry map[string]interface{}
		if json.Unmarshal([]byte(line), &entry) != nil || entry["app.log_type"] != "access" {
			continue
		}
		accessCount++
		if entry["app.response_code"] != "CORE-1001" || entry["app.class"] != "FAILED" || entry["log.level"] != "info" {
			t.Fatalf("unexpected auth-rejection access log: %#v", entry)
		}
	}
	if accessCount != 1 {
		t.Fatalf("expected exactly one gRPC access log, got %d; output=%s", accessCount, output)
	}
}

func directServiceDescription(service string, call func(context.Context) error) *grpc.ServiceDesc {
	return &grpc.ServiceDesc{
		ServiceName: service,
		HandlerType: (*interface{})(nil),
		Methods: []grpc.MethodDesc{{
			MethodName: "Call",
			Handler: func(_ interface{}, ctx context.Context, _ func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
				handler := func(context.Context, interface{}) (interface{}, error) { return nil, call(ctx) }
				if interceptor == nil {
					return handler(ctx, nil)
				}
				return interceptor(ctx, nil, &grpc.UnaryServerInfo{FullMethod: "/" + service + "/Call"}, handler)
			},
		}},
	}
}
