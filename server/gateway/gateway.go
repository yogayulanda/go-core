package gateway

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"

	"github.com/yogayulanda/go-core/app"
	"github.com/yogayulanda/go-core/logger"
)

type Gateway struct {
	server      *http.Server
	tlsEnabled  bool
	tlsCertFile string
	tlsKeyFile  string
	log         logger.Logger
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
	if cfg.HTTP.PprofEnabled {
		if err := registerPprofEndpoints(mux); err != nil {
			return nil, fmt.Errorf("failed to register pprof endpoints: %w", err)
		}
	}

	var handler http.Handler = mux
	handler = withHTTPRequestID(handler)
	handler = withHTTPMetrics(application, handler)
	handler = withCORS(handler)
	handler = otelhttp.NewHandler(handler, "http_gateway")
	handler = withPanicRecovery(application, handler)

	httpServer := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.HTTP.Port),
		Handler:      handler,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	application.Lifecycle().Register(func(ctx context.Context) error {
		log.LogService(ctx, logger.ServiceLog{
			Operation: "http_gateway",
			Status:    "shutdown_requested",
		})
		return httpServer.Shutdown(ctx)
	})

	return &Gateway{
		server:      httpServer,
		tlsEnabled:  cfg.HTTP.TLSEnabled,
		tlsCertFile: cfg.HTTP.TLSCertFile,
		tlsKeyFile:  cfg.HTTP.TLSKeyFile,
		log:         log,
	}, nil
}

// Start runs the HTTP server.
func (g *Gateway) Start() error {
	g.log.LogService(context.Background(), logger.ServiceLog{
		Operation: "http_gateway",
		Status:    "started",
		Metadata: map[string]interface{}{
			"address": g.server.Addr,
			"tls":     g.tlsEnabled,
		},
	})
	if g.tlsEnabled {
		return g.server.ListenAndServeTLS(g.tlsCertFile, g.tlsKeyFile)
	}
	return g.server.ListenAndServe()
}
