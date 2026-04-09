package grpc

import (
	"context"
	"fmt"
	"net"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/keepalive"

	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"

	"github.com/yogayulanda/go-core/app"
	"github.com/yogayulanda/go-core/logger"
	"github.com/yogayulanda/go-core/security"
)

type Server struct {
	server *grpc.Server
	lis    net.Listener
	app    *app.App
}

func (s *Server) Name() string {
	return "grpc_server"
}

// New creates a new gRPC server with interceptors.
func New(application *app.App) (*Server, error) {
	if application == nil {
		return nil, fmt.Errorf("application is nil")
	}

	cfg := application.Config()
	if cfg == nil {
		return nil, fmt.Errorf("application config is nil")
	}
	if application.Lifecycle() == nil {
		return nil, fmt.Errorf("application lifecycle is nil")
	}

	log := application.Logger()

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.GRPC.Port))
	if err != nil {
		return nil, fmt.Errorf("failed to listen on port %d: %w", cfg.GRPC.Port, err)
	}

	kaParams := keepalive.ServerParameters{
		MaxConnectionIdle:     5 * time.Minute,
		MaxConnectionAge:      30 * time.Minute,
		MaxConnectionAgeGrace: 5 * time.Minute,
		Time:                  2 * time.Hour,
		Timeout:               20 * time.Second,
	}

	kaPolicy := keepalive.EnforcementPolicy{
		MinTime:             5 * time.Minute,
		PermitWithoutStream: false,
	}

	authVerifier, err := security.NewInternalJWTVerifier(cfg.Auth.InternalJWT)
	if err != nil {
		return nil, fmt.Errorf("init auth verifier failed: %w", err)
	}
	logAuthConfig(context.Background(), log, authVerifier)

	serverOpts := []grpc.ServerOption{
		grpc.ChainUnaryInterceptor(
			recoveryInterceptor(log),
			requestIDInterceptor(),
			authInterceptorWithLogger(authVerifier, log),
			loggingInterceptor(application),
			metricsInterceptor(application),
		),
		grpc.StatsHandler(otelgrpc.NewServerHandler()),
		grpc.KeepaliveParams(kaParams),
		grpc.KeepaliveEnforcementPolicy(kaPolicy),
		grpc.MaxRecvMsgSize(10 * 1024 * 1024),
		grpc.MaxSendMsgSize(10 * 1024 * 1024),
	}

	if cfg.GRPC.TLSEnabled {
		creds, err := credentials.NewServerTLSFromFile(cfg.GRPC.TLSCertFile, cfg.GRPC.TLSKeyFile)
		if err != nil {
			return nil, fmt.Errorf("init grpc tls failed: %w", err)
		}
		serverOpts = append(serverOpts, grpc.Creds(creds))
	}

	server := grpc.NewServer(serverOpts...)

	s := &Server{
		server: server,
		lis:    lis,
		app:    application,
	}

	application.Lifecycle().Register(func(ctx context.Context) error {
		log.LogService(ctx, appLog("grpc_server", "shutdown_requested", 0, "", nil))
		err := gracefulStopWithTimeout(ctx, server)
		if err != nil {
			log.LogService(ctx, appLog("grpc_server", "failed", 0, "graceful_stop_timeout", map[string]interface{}{
				"error": err.Error(),
			}))
		}
		return err
	})

	return s, nil
}

// Register allows service layer to register gRPC handlers.
func (s *Server) Register(registerFunc func(*grpc.Server)) {
	registerFunc(s.server)
}

// Start runs the gRPC server.
func (s *Server) Start() error {
	log := s.app.Logger()
	log.LogService(context.Background(), appLog("grpc_server", "started", 0, "", map[string]interface{}{
		"address": s.lis.Addr().String(),
	}))
	return s.server.Serve(s.lis)
}

func appLog(operation string, status string, durationMs int64, errorCode string, metadata map[string]interface{}) logger.ServiceLog {
	return logger.ServiceLog{
		Operation:  operation,
		Status:     status,
		DurationMs: durationMs,
		ErrorCode:  errorCode,
		Metadata:   metadata,
	}
}

type grpcStopper interface {
	GracefulStop()
	Stop()
}

func gracefulStopWithTimeout(ctx context.Context, server grpcStopper) error {
	done := make(chan struct{})
	go func() {
		server.GracefulStop()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		server.Stop()
		<-done
		return ctx.Err()
	}
}
