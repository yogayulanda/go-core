package logger

import (
	"context"
	"os"
	"strings"
	"time"

	"github.com/yogayulanda/go-core/observability"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type zapLogger struct {
	l *zap.Logger
}

func New(serviceName string, level string) (Logger, error) {
	parsedLevel, err := parseLevel(level)
	if err != nil {
		return nil, err
	}
	logLocation := resolveLogLocation(os.Getenv("LOG_TIMEZONE"))

	isDev := isDevEnvironment(os.Getenv("APP_ENV"))
	encoding := "json"
	encodeLevel := zapcore.LowercaseLevelEncoder
	if isDev {
		encoding = "console"
		encodeLevel = zapcore.CapitalColorLevelEncoder
	}

	cfg := zap.Config{
		Level:       zap.NewAtomicLevelAt(parsedLevel),
		Development: isDev,
		Encoding:    encoding,
		EncoderConfig: zapcore.EncoderConfig{
			TimeKey:       "timestamp",
			LevelKey:      "level",
			MessageKey:    "message",
			CallerKey:     "caller",
			StacktraceKey: "stacktrace",
			EncodeTime:    timeEncoder(logLocation),
			EncodeLevel:   encodeLevel,
			EncodeCaller:  zapcore.ShortCallerEncoder,
		},
		OutputPaths:      []string{"stdout"},
		ErrorOutputPaths: []string{"stderr"},
	}

	base, err := cfg.Build()
	if err != nil {
		return nil, err
	}

	base = base.With(
		zap.String("service", serviceName),
		zap.String("timezone", logLocation.String()),
	)

	return &zapLogger{l: base}, nil
}

func isDevEnvironment(env string) bool {
	switch strings.ToLower(strings.TrimSpace(env)) {
	case "dev", "local", "development":
		return true
	default:
		return false
	}
}

func timeEncoder(location *time.Location) zapcore.TimeEncoder {
	if location == nil {
		location = time.UTC
	}
	return func(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
		enc.AppendString(t.In(location).Format(time.RFC3339))
	}
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

func (z *zapLogger) WithComponent(component string) Logger {
	return &zapLogger{
		l: z.l.With(zap.String("component", component)),
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
	zapFields := make([]zap.Field, 0, len(fields)+3)
	hasRequestIDField := false

	for _, f := range fields {
		if strings.EqualFold(strings.TrimSpace(f.Key), "request_id") {
			hasRequestIDField = true
			break
		}
	}

	// Inject trace_id and span_id automatically if exists
	if span := trace.SpanFromContext(ctx); span != nil {
		sc := span.SpanContext()
		if sc.IsValid() {
			zapFields = append(zapFields,
				zap.String("trace_id", sc.TraceID().String()),
				zap.String("span_id", sc.SpanID().String()),
			)
		}
	}

	if !hasRequestIDField {
		if requestID := observability.GetRequestID(ctx); requestID != "" {
			zapFields = append(zapFields, zap.String("request_id", requestID))
		}
	}

	hasTransactionIDField := false
	for _, f := range fields {
		if strings.EqualFold(strings.TrimSpace(f.Key), FieldTransactionID) {
			hasTransactionIDField = true
			break
		}
	}
	if !hasTransactionIDField {
		if txID := observability.GetTransactionID(ctx); txID != "" {
			zapFields = append(zapFields, zap.String(FieldTransactionID, txID))
		}
	}

	for _, f := range fields {
		zapFields = append(zapFields, zap.Any(f.Key, sanitizeFieldValue(f.Key, f.Value)))
	}

	return zapFields
}
