package grpcclient

import (
	"context"
	"fmt"
	"net"
	"sync"
	"testing"
	"time"

	errdetails "google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/yogayulanda/go-core/logger"
	"github.com/yogayulanda/go-core/observability"
)

type testService interface {
	Call(context.Context, *emptypb.Empty) (*emptypb.Empty, error)
}

type testGRPCServer struct {
	mu       sync.Mutex
	calls    int
	metadata metadata.MD
	deadline time.Time
}

func (s *testGRPCServer) Call(ctx context.Context, _ *emptypb.Empty) (*emptypb.Empty, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.calls++
	s.metadata, _ = metadata.FromIncomingContext(ctx)
	s.deadline, _ = ctx.Deadline()
	return nil, status.Error(codes.Unavailable, "temporarily unavailable")
}

func testCallHandler(server interface{}, ctx context.Context, decode func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	request := new(emptypb.Empty)
	if err := decode(request); err != nil {
		return nil, err
	}
	handler := func(ctx context.Context, request interface{}) (interface{}, error) {
		service, ok := server.(testService)
		if !ok {
			return nil, fmt.Errorf("unexpected gRPC service type %T", server)
		}
		decodedRequest, ok := request.(*emptypb.Empty)
		if !ok {
			return nil, fmt.Errorf("unexpected gRPC request type %T", request)
		}
		response, err := service.Call(ctx, decodedRequest)
		if err != nil {
			return nil, err
		}
		return response, nil
	}
	if interceptor == nil {
		return handler(ctx, request)
	}
	return interceptor(ctx, request, &grpc.UnaryServerInfo{Server: server, FullMethod: "/test.Service/Call"}, handler)
}

var testServiceDesc = grpc.ServiceDesc{
	ServiceName: "test.Service",
	HandlerType: (*testService)(nil),
	Methods:     []grpc.MethodDesc{{MethodName: "Call", Handler: testCallHandler}},
}

func TestClient_PropagatesIdentityUsesDeadlineAndDoesNotRetry(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	server := grpc.NewServer()
	service := &testGRPCServer{}
	server.RegisterService(&testServiceDesc, service)
	go func() { _ = server.Serve(listener) }()
	defer server.Stop()

	const propagatedLabel = "grpcclient_test_label"
	if err := observability.RegisterPropagatedLabel(propagatedLabel); err != nil {
		t.Fatal(err)
	}
	ctx := observability.EnsureFieldBag(context.Background())
	ctx = observability.WithRequestID(ctx, "req-grpc-client")
	if err := observability.SetUserID(ctx, "user-grpc"); err != nil {
		t.Fatal(err)
	}
	if err := observability.SetLabel(ctx, propagatedLabel, "inside"); err != nil {
		t.Fatal(err)
	}
	log := &testLogger{}
	client, err := NewClient(listener.Addr().String(), log,
		WithDialOptions(grpc.WithDefaultServiceConfig(`{"methodConfig":[{"name":[{"service":"test.Service"}],"retryPolicy":{"maxAttempts":4,"initialBackoff":"0.01s","maxBackoff":"0.01s","backoffMultiplier":1,"retryableStatusCodes":["UNAVAILABLE"]}}]}`)),
	)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	callErr := client.Conn().Invoke(ctx, "/test.Service/Call", &emptypb.Empty{}, &emptypb.Empty{})
	if status.Code(callErr) != codes.Unavailable {
		t.Fatalf("expected server UNAVAILABLE, got %v", callErr)
	}
	service.mu.Lock()
	defer service.mu.Unlock()
	if service.calls != 1 {
		t.Fatalf("UNAVAILABLE call was retried %d times", service.calls)
	}
	if service.metadata.Get("x-request-id")[0] != "req-grpc-client" || service.metadata.Get("x-user-id")[0] != "user-grpc" || service.metadata.Get("x-label-" + propagatedLabel)[0] != "inside" {
		t.Fatalf("outgoing identity metadata missing: %#v", service.metadata)
	}
	if service.deadline.IsZero() || time.Until(service.deadline) > 30*time.Second || time.Until(service.deadline) < 29*time.Second {
		t.Fatalf("default call deadline missing or outside expected window: %v", service.deadline)
	}
	if len(log.services) != 1 || log.services[0].Operation != "grpc_client_call" {
		t.Fatalf("expected exactly one call log, got %#v", log.services)
	}
}

func TestResponseCodeAndErrorInfoHelpers(t *testing.T) {
	if got := ResponseCodeFromHeader(metadata.Pairs("x-response-code", "CORE-9001")); got != "CORE-9001" {
		t.Fatalf("unexpected response code: %q", got)
	}
	st, err := status.New(codes.InvalidArgument, "invalid").WithDetails(&errdetails.ErrorInfo{
		Reason: "CORE-1000",
		Domain: "go-core",
		Metadata: map[string]string{
			"class":     "FAILED",
			"finality":  "FINAL",
			"retryable": "false",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	info, ok := ErrorInfo(st.Err())
	if !ok || info.Code != "CORE-1000" || info.Class != "FAILED" || info.Finality != "FINAL" || info.Retryable {
		t.Fatalf("unexpected ErrorInfo: %#v, ok=%v", info, ok)
	}
}

type testLogger struct {
	services []logger.ServiceLog
}

func (*testLogger) Info(context.Context, string, ...logger.Field)  {}
func (*testLogger) Error(context.Context, string, ...logger.Field) {}
func (*testLogger) Debug(context.Context, string, ...logger.Field) {}
func (*testLogger) Warn(context.Context, string, ...logger.Field)  {}
func (l *testLogger) LogService(_ context.Context, service logger.ServiceLog) {
	l.services = append(l.services, service)
}
func (*testLogger) LogDB(context.Context, logger.DBLog)                   {}
func (*testLogger) LogEvent(context.Context, logger.EventLog)             {}
func (*testLogger) LogTransaction(context.Context, logger.TransactionLog) {}
func (l *testLogger) WithComponent(string) logger.Logger                  { return l }
