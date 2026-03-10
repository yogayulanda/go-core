package gateway

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"

	"github.com/yogayulanda/go-core/app"
)

type Gateway struct {
	server      *http.Server
	tlsEnabled  bool
	tlsCertFile string
	tlsKeyFile  string
}

func (g *Gateway) Name() string {
	return "http_gateway"
}

// New initializes HTTP gateway server.
func New(application *app.App, registerFunc func(ctx context.Context, mux *runtime.ServeMux) error) (*Gateway, error) {
	if application == nil {
		return nil, fmt.Errorf("application is nil")
	}
	if registerFunc == nil {
		return nil, fmt.Errorf("registerFunc is nil")
	}

	cfg := application.Config()
	if cfg == nil {
		return nil, fmt.Errorf("application config is nil")
	}
	if application.Lifecycle() == nil {
		return nil, fmt.Errorf("application lifecycle is nil")
	}

	log := application.Logger()

	mux := runtime.NewServeMux(
		runtime.WithErrorHandler(customErrorHandler(application)),
	)

	ctx := context.Background()

	if err := registerFunc(ctx, mux); err != nil {
		return nil, fmt.Errorf("failed to register gateway handlers: %w", err)
	}

	if err := registerHealthEndpoints(mux, application); err != nil {
		return nil, fmt.Errorf("failed to register health endpoints: %w", err)
	}
	if err := registerVersionEndpoint(mux); err != nil {
		return nil, fmt.Errorf("failed to register version endpoint: %w", err)
	}
	if err := registerMetricsEndpoint(mux); err != nil {
		return nil, fmt.Errorf("failed to register metrics endpoint: %w", err)
	}

	httpServer := &http.Server{
		Addr: fmt.Sprintf(":%d", cfg.HTTP.Port),
		Handler: withHTTPMetrics(
			application,
			withHTTPRequestID(mux),
		),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	application.Lifecycle().Register(func(ctx context.Context) error {
		log.Info(ctx, "shutting down HTTP gateway")
		return httpServer.Shutdown(ctx)
	})

	return &Gateway{
		server:      httpServer,
		tlsEnabled:  cfg.HTTP.TLSEnabled,
		tlsCertFile: cfg.HTTP.TLSCertFile,
		tlsKeyFile:  cfg.HTTP.TLSKeyFile,
	}, nil
}

// Start runs the HTTP server.
func (g *Gateway) Start() error {
	if g.tlsEnabled {
		return g.server.ListenAndServeTLS(g.tlsCertFile, g.tlsKeyFile)
	}
	return g.server.ListenAndServe()
}
