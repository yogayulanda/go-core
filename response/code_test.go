package response

import (
	"context"
	"net"
	"net/http"
	"testing"

	"github.com/yogayulanda/go-core/observability"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/test/bufconn"
	"google.golang.org/protobuf/types/known/emptypb"
)

func TestCodeFromHeaders(t *testing.T) {
	if got := CodeFromHeader(metadata.Pairs("x-response-code", "CORE-0000")); got != "CORE-0000" {
		t.Fatalf("unexpected gRPC response code: %q", got)
	}
	header := http.Header{"Grpc-Metadata-X-Response-Code": []string{"CORE-1000"}}
	if got := CodeFromHTTPHeader(header); got != "CORE-1000" {
		t.Fatalf("unexpected HTTP gateway response code: %q", got)
	}
}

func TestSetCodeRecordsRegisteredAndUnknownDefinition(t *testing.T) {
	for _, test := range []struct {
		input, wantCode, wantClass, wantOriginal string
	}{
		{input: "CORE-0000", wantCode: "CORE-0000", wantClass: "SUCCESS"},
		{input: "CUSTOM-UNKNOWN", wantCode: "CORE-9000", wantClass: "ERROR", wantOriginal: "CUSTOM-UNKNOWN"},
	} {
		listener := bufconn.Listen(1024 * 1024)
		server := grpc.NewServer()
		infoCh := make(chan [3]string, 1)
		server.RegisterService(&grpc.ServiceDesc{
			ServiceName: "response.Code",
			HandlerType: (*interface{})(nil),
			Methods: []grpc.MethodDesc{{
				MethodName: "Set",
				Handler: func(_ interface{}, ctx context.Context, _ func(interface{}) error, _ grpc.UnaryServerInterceptor) (interface{}, error) {
					ctx = observability.EnsureFieldBag(ctx)
					if err := SetCode(ctx, test.input); err != nil {
						return nil, err
					}
					infoCh <- func() [3]string { a, b, c := observability.ResponseInfo(ctx); return [3]string{a, b, c} }()
					return &emptypb.Empty{}, nil
				},
			}},
		}, nil)
		go func() { _ = server.Serve(listener) }()
		conn, err := grpc.NewClient("passthrough:///bufconn", grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) { return listener.Dial() }), grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			t.Fatalf("dial bufconn: %v", err)
		}
		var headers metadata.MD
		if err := conn.Invoke(context.Background(), "/response.Code/Set", &emptypb.Empty{}, &emptypb.Empty{}, grpc.Header(&headers)); err != nil {
			t.Fatalf("invoke SetCode: %v", err)
		}
		got := <-infoCh
		if got != [3]string{test.wantCode, test.wantClass, test.wantOriginal} {
			t.Fatalf("unexpected response info: %v", got)
		}
		if gotCode := CodeFromHeader(headers); gotCode != test.wantCode {
			t.Fatalf("unexpected response header code: %q", gotCode)
		}
		_ = conn.Close()
		server.Stop()
		_ = listener.Close()
	}
}
