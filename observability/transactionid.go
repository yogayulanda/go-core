package observability

import (
	"context"
)

const transactionIDKey contextKey = "transaction_id"

// WithTransactionID injects transaction ID into context.
func WithTransactionID(ctx context.Context, transactionID string) context.Context {
	return context.WithValue(ctx, transactionIDKey, transactionID)
}

// GetTransactionID extracts transaction ID from context.
func GetTransactionID(ctx context.Context) string {
	if val, ok := ctx.Value(transactionIDKey).(string); ok {
		return val
	}
	return ""
}
