package gateway

import (
	"context"
	"fmt"
	"net/http"
	"strings"
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
		runtime.WithIncomingHeaderMatcher(incomingHeaderMatcher),
		runtime.WithOutgoingHeaderMatcher(outgoingHeaderMatcher),
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
		if cfg.App.Environment == "prod" || cfg.App.Environment == "production" {
			return nil, fmt.Errorf("pprof is forbidden in production")
		}
		if err := registerPprofEndpoints(mux); err != nil {
			return nil, fmt.Errorf("failed to register pprof endpoints: %w", err)
		}
	}

	var handler http.Handler = mux
	handler = withHTTPMetrics(application, handler)
	handler = withSuccessEnvelope(handler)
	handler = withCORS(handler, cfg.HTTP.CORSAllowedOrigins, cfg.HTTP.CORSAllowedHeaders)
	handler = otelhttp.NewHandler(handler, "http_gateway")
	handler = withSignatureValidation(application, handler)
	handler = withRequestBodyLimit(application, cfg.HTTP.MaxBodyBytes, handler)
	handler = withPanicRecovery(application, handler)
	handler = withHTTPRequestIdentity(application, cfg.HTTP.TrustedProxyHops, handler)

	httpServer := &http.Server{
		Addr:              fmt.Sprintf(":%d", cfg.HTTP.Port),
		Handler:           handler,
		ReadHeaderTimeout: cfg.HTTP.ReadHeaderTimeout,
		ReadTimeout:       cfg.HTTP.ReadTimeout,
		WriteTimeout:      cfg.HTTP.WriteTimeout,
		IdleTimeout:       cfg.HTTP.IdleTimeout,
	}

	application.RegisterServerShutdown("http_gateway", func(ctx context.Context) error {
		startedAt := time.Now()
		log.LogService(ctx, logger.ServiceLog{
			Operation: "http_gateway",
			Status:    "shutdown_requested",
		})
		err := httpServer.Shutdown(ctx)
		if err != nil {
			application.LogInflightCut()
			_ = httpServer.Close()
		}
		application.LogShutdownPhase("http_gateway", startedAt, err)
		return err
	})

	return &Gateway{
		server:      httpServer,
		tlsEnabled:  cfg.HTTP.TLSEnabled,
		tlsCertFile: cfg.HTTP.TLSCertFile,
		tlsKeyFile:  cfg.HTTP.TLSKeyFile,
		log:         log,
	}, nil
}

func incomingHeaderMatcher(key string) (string, bool) {
	lower := strings.ToLower(key)
	switch {
	case lower == "x-request-id", lower == "x-user-id", lower == "x-core-client-ip", lower == "x-core-user-agent", lower == "x-core-request-id-replaced", strings.HasPrefix(lower, "x-label-"):
		return lower, true
	default:
		return runtime.DefaultHeaderMatcher(key)
	}
}

func outgoingHeaderMatcher(key string) (string, bool) {
	if strings.EqualFold(key, "x-request-id") {
		return "X-Request-ID", true
	}
	if strings.EqualFold(key, "x-response-code") || strings.EqualFold(key, "grpc-metadata-x-response-code") {
		return "X-Response-Code", true
	}
	return runtime.DefaultHeaderMatcher(key)
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
