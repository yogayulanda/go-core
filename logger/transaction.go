package logger

import (
	"context"
	"fmt"

	"github.com/yogayulanda/go-core/observability"
)

// TransactionLog represents the platform-standard monitoring contract
// for transaction-oriented services.
//
// Purpose:
// - Transaction flow monitoring
// - Grafana dashboard
// - Success rate tracking
// - Latency tracking
// - Alerting by error_code
//
// Top-level fields are considered a stable monitoring contract.
// Additional service-specific details must go into Metadata.
//
// Identity fields are resolved from the request context field bag.
//
// ⚠ DO NOT log sensitive information such as:
// - password
// - token
// - card number
// - full request body
type TransactionLog struct {
	Feature        string
	Product        string
	Status         string
	PreviousStatus string
	ResponseCode   string
	Amount         interface{}
	DurationMs     int64
}

func (z *zapLogger) LogTransaction(ctx context.Context, tx TransactionLog) {
	fields := []Field{
		{Key: "app.log_type", Value: "transaction"},
		{Key: "app.transaction.feature", Value: tx.Feature},
		{Key: "app.transaction.product", Value: tx.Product},
		{Key: "app.transaction.status", Value: tx.Status},
		{Key: "app.transaction.previous_status", Value: tx.PreviousStatus},
		{Key: "app.transaction.response_code", Value: tx.ResponseCode},
		{Key: "app.transaction.duration_ms", Value: tx.DurationMs},
	}
	if tx.Amount != nil {
		fields = append(fields, Field{Key: "app.transaction.amount", Value: tx.Amount})
	}
	z.Info(ctx, fmt.Sprintf("transaction %s %s %s -> %s %s %dms", tx.Feature, observability.GetGTID(ctx), tx.PreviousStatus, tx.Status, tx.ResponseCode, tx.DurationMs), fields...)
}
