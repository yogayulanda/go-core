package observability

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/yogayulanda/go-core/config"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	"google.golang.org/grpc/credentials"
)

func InitTracing(ctx context.Context, cfg *config.Config) (func(context.Context) error, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is nil")
	}

	if cfg.Observability.OTLPEndpoint == "" {
		// Tracing disabled
		return func(context.Context) error { return nil }, nil
	}

	exporterOpts := []otlptracegrpc.Option{
		otlptracegrpc.WithEndpoint(cfg.Observability.OTLPEndpoint),
	}

	if cfg.Observability.OTLPInsecure {
		exporterOpts = append(exporterOpts, otlptracegrpc.WithInsecure())
	} else {
		tlsCfg, err := buildOTLPTLSConfig(cfg.Observability.OTLPCACertFile)
		if err != nil {
			return nil, err
		}
		exporterOpts = append(exporterOpts, otlptracegrpc.WithTLSCredentials(credentials.NewTLS(tlsCfg)))
	}

	exporter, err := otlptracegrpc.New(ctx, exporterOpts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create otlp exporter: %w", err)
	}

	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName(cfg.App.ServiceName),
			semconv.DeploymentEnvironment(cfg.App.Environment),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	sampler := sdktrace.ParentBased(
		sdktrace.TraceIDRatioBased(cfg.Observability.TraceSamplingRatio),
	)

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithSampler(sampler),
		sdktrace.WithResource(res),
	)

	otel.SetTracerProvider(tp)

	otel.SetTextMapPropagator(
		propagation.NewCompositeTextMapPropagator(
			propagation.TraceContext{},
			propagation.Baggage{},
		),
	)

	return func(ctx context.Context) error {
		ctxTimeout, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		return tp.Shutdown(ctxTimeout)
	}, nil
}

func buildOTLPTLSConfig(caCertFile string) (*tls.Config, error) {
	cfg := &tls.Config{
		MinVersion: tls.VersionTLS12,
	}

	file := strings.TrimSpace(caCertFile)
	if file == "" {
		return cfg, nil
	}

	raw, err := os.ReadFile(file)
	if err != nil {
		return nil, fmt.Errorf("read OTLP CA cert file failed: %w", err)
	}
	pool := x509.NewCertPool()
	if ok := pool.AppendCertsFromPEM(raw); !ok {
		return nil, fmt.Errorf("invalid OTLP CA cert file: no certificate found")
	}
	cfg.RootCAs = pool
	return cfg, nil
}
