package grpcclient

import (
	"context"
	"fmt"
	"time"

	coreerrors "github.com/yogayulanda/go-core/errors"
	"github.com/yogayulanda/go-core/logger"
	"github.com/yogayulanda/go-core/observability"
	"github.com/yogayulanda/go-core/response"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type Client struct {
	conn *grpc.ClientConn
}

func NewClient(target string, log logger.Logger, options ...Option) (*Client, error) {
	if target == "" {
		return nil, fmt.Errorf("grpc client target is empty")
	}
	if log == nil {
		return nil, fmt.Errorf("grpc client logger is nil")
	}
	cfg := defaultOptions()
	for _, option := range options {
		if option != nil {
			option(&cfg)
		}
	}
	if cfg.ServiceName == "" {
		return nil, fmt.Errorf("grpc client service name is empty")
	}
	if cfg.DefaultDeadline <= 0 {
		return nil, fmt.Errorf("grpc client default deadline must be positive")
	}
	if cfg.TransportCredentials == nil {
		return nil, fmt.Errorf("grpc client transport credentials are nil")
	}
	metrics := observability.NewMetrics()
	dialOptions := append([]grpc.DialOption{
		grpc.WithTransportCredentials(cfg.TransportCredentials),
		grpc.WithStatsHandler(otelgrpc.NewClientHandler()),
		grpc.WithKeepaliveParams(cfg.Keepalive),
		grpc.WithDisableRetry(),
		grpc.WithChainUnaryInterceptor(clientInterceptor(log, cfg, metrics)),
	}, cfg.DialOptions...)
	conn, err := grpc.NewClient(target, dialOptions...)
	if err != nil {
		return nil, fmt.Errorf("create grpc client for %q: %w", target, err)
	}
	return &Client{conn: conn}, nil
}

func Dial(ctx context.Context, target string, log logger.Logger, options ...Option) (*Client, error) {
	if ctx == nil {
		return nil, fmt.Errorf("grpc dial context is nil")
	}
	client, err := NewClient(target, log, options...)
	if err != nil {
		return nil, err
	}
	client.conn.Connect()
	for {
		state := client.conn.GetState()
		if state == connectivity.Ready {
			return client, nil
		}
		if !client.conn.WaitForStateChange(ctx, state) {
			_ = client.Close()
			return nil, fmt.Errorf("dial grpc target %q: %w", target, ctx.Err())
		}
	}
}

func (c *Client) Conn() *grpc.ClientConn { return c.conn }

func (c *Client) Close() error { return c.conn.Close() }

func ResponseCodeFromHeader(headers metadata.MD) string {
	return response.CodeFromHeader(headers)
}

func ErrorInfo(err error) (coreerrors.GRPCErrorInfo, bool) {
	return coreerrors.InfoFromGRPC(err)
}

func clientInterceptor(log logger.Logger, options Options, metrics *observability.Metrics) grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, request, reply interface{}, conn *grpc.ClientConn, invoke grpc.UnaryInvoker, callOptions ...grpc.CallOption) error {
		ctx = observability.EnsureFieldBag(ctx)
		ctx, cancel := withDefaultDeadline(ctx, options.DefaultDeadline)
		defer cancel()
		ctx = outgoingIdentityContext(ctx)
		started := time.Now()
		err := invoke(ctx, method, request, reply, conn, callOptions...)
		duration := time.Since(started)
		statusName := "success"
		grpcCode := codes.OK.String()
		if err != nil {
			grpcCode = status.Code(err).String()
			statusName = "failed"
		}
		log.LogService(ctx, logger.ServiceLog{
			Operation:  "grpc_client_call",
			Status:     statusName,
			DurationMs: duration.Milliseconds(),
			ErrorCode:  grpcCode,
			Metadata:   map[string]interface{}{"method": method, "grpc_code": grpcCode},
		})
		metrics.ServiceTotal.WithLabelValues(options.ServiceName, method, grpcCode).Inc()
		metrics.ServiceDuration.WithLabelValues(options.ServiceName, method).Observe(duration.Seconds())
		return err
	}
}

func withDefaultDeadline(ctx context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	if _, exists := ctx.Deadline(); exists {
		return ctx, func() {}
	}
	return context.WithTimeout(ctx, timeout)
}

func outgoingIdentityContext(ctx context.Context) context.Context {
	md, _ := metadata.FromOutgoingContext(ctx)
	md = md.Copy()
	setIfMissing := func(key, value string) {
		if value != "" && len(md.Get(key)) == 0 {
			md.Set(key, value)
		}
	}
	setIfMissing("x-request-id", observability.GetRequestID(ctx))
	setIfMissing("x-user-id", observability.GetUserID(ctx))
	labels := observability.Labels(ctx)
	for _, name := range observability.PropagatedLabels() {
		setIfMissing(observability.LabelMetadataPrefix+name, labels[name])
	}
	return metadata.NewOutgoingContext(ctx, md)
}
