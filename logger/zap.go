package logger

import (
	"context"
	"io"
	"os"
	"strings"
	"time"

	"github.com/yogayulanda/go-core/observability"
	"github.com/yogayulanda/go-core/version"
	"go.elastic.co/ecszap"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type zapLogger struct {
	l      *zap.Logger
	config Config
}

type Options struct {
	Environment string
	Config      Config
	Output      io.Writer
}

type Config struct {
	MaskKeys             []string
	RequestStart         bool
	SlowQueryThreshold   time.Duration
	SlowRequestThreshold time.Duration
}

func New(serviceName string, level string) (Logger, error) {
	return NewWithOptions(serviceName, level, Options{})
}

func NewWithOptions(serviceName string, level string, options Options) (Logger, error) {
	parsedLevel, err := parseLevel(level)
	if err != nil {
		return nil, err
	}
	output := options.Output
	if output == nil {
		output = os.Stdout
	}
	writeSyncer := zapcore.AddSync(output)
	var core zapcore.Core
	if isLocalEnvironment(options.Environment) {
		core = newConsoleCore(writeSyncer, parsedLevel, isTTY(output))
	} else {
		encoderConfig := ecszap.NewDefaultEncoderConfig()
		encoderConfig.EnableStackTrace = false
		jsonConfig := encoderConfig.ToZapCoreEncoderConfig()
		jsonConfig.EncodeTime = ecsTimeEncoder
		core = ecszap.WrapCore(zapcore.NewCore(zapcore.NewJSONEncoder(jsonConfig), orderedECSWriter{output: writeSyncer}, parsedLevel))
	}
	base := zap.New(core).With(
		zap.String("service.name", serviceName),
		zap.String("service.environment", options.Environment),
		zap.String("service.version", version.Version),
	)
	return &zapLogger{l: base, config: options.Config}, nil
}

func ecsTimeEncoder(t time.Time, encoder zapcore.PrimitiveArrayEncoder) {
	encoder.AppendString(t.UTC().Format("2006-01-02T15:04:05.000Z"))
}

func resolveLogLocation(raw string) *time.Location {
	tz := strings.TrimSpace(raw)
	if tz == "" {
		return time.UTC
	}
	loc, err := time.LoadLocation(tz)
	if err != nil {
		return time.UTC
	}
	return loc
}

func isLocalEnvironment(environment string) bool {
	return strings.TrimSpace(environment) == "local"
}

func (z *zapLogger) WithComponent(component string) Logger {
	return &zapLogger{
		l:      z.l.With(zap.String("labels.component", component)),
		config: z.config,
	}
}

func (z *zapLogger) Info(ctx context.Context, msg string, fields ...Field) {
	z.l.Info(msg, z.buildFields(ctx, fields...)...)
}

func (z *zapLogger) Error(ctx context.Context, msg string, fields ...Field) {
	z.l.Error(msg, z.buildFields(ctx, fields...)...)
}

func (z *zapLogger) Debug(ctx context.Context, msg string, fields ...Field) {
	z.l.Debug(msg, z.buildFields(ctx, fields...)...)
}

func (z *zapLogger) Warn(ctx context.Context, msg string, fields ...Field) {
	z.l.Warn(msg, z.buildFields(ctx, fields...)...)
}

func (z *zapLogger) buildFields(ctx context.Context, fields ...Field) []zap.Field {
	zapFields := make([]zap.Field, 0, len(fields)+8)
	hasRequestIDField := false

	for _, f := range fields {
		if strings.EqualFold(strings.TrimSpace(f.Key), "request_id") {
			hasRequestIDField = true
			break
		}
	}

	// Inject ECS trace fields automatically when a span is present.
	if span := trace.SpanFromContext(ctx); span != nil {
		sc := span.SpanContext()
		if sc.IsValid() {
			zapFields = append(zapFields,
				zap.String("trace.id", sc.TraceID().String()),
				zap.String("span.id", sc.SpanID().String()),
			)
		}
	}

	if !hasRequestIDField {
		if requestID := observability.GetRequestID(ctx); requestID != "" {
			zapFields = append(zapFields, zap.String("labels.request_id", requestID))
		}
	}

	if gtid := observability.GetGTID(ctx); gtid != "" {
		zapFields = append(zapFields, zap.String("labels.gtid", gtid))
	}
	if userID := observability.GetUserID(ctx); userID != "" {
		zapFields = append(zapFields, zap.String("user.id", userID))
	}
	for key, value := range observability.Labels(ctx) {
		zapFields = append(zapFields, zap.String("labels."+key, value))
	}

	for _, f := range fields {
		if f.Key == "request_id" {
			f.Key = "labels.request_id"
		}
		if f.Key == "transaction_id" {
			f.Key = "labels.gtid"
		}
		zapFields = append(zapFields, zap.Any(f.Key, sanitizeFieldValueWithKeys(f.Key, f.Value, z.config.MaskKeys)))
	}

	return zapFields
}
