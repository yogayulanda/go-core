package observability

import (
	"context"

	"github.com/google/uuid"
)

type contextKey string

const requestIDKey contextKey = "request_id"

// WithRequestID injects request ID into context.
func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, requestIDKey, requestID)
}

// GetRequestID extracts request ID from context.
func GetRequestID(ctx context.Context) string {
	if val, ok := ctx.Value(requestIDKey).(string); ok {
		return val
	}
	return ""
}

// GenerateRequestID creates new UUID.
func GenerateRequestID() string {
	return uuid.NewString()
}
