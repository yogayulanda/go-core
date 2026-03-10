package logger

import "context"

// TransactionLog represents business transaction monitoring log.
//
// Purpose:
// - Business operation monitoring
// - Grafana dashboard
// - Success rate tracking
// - Latency tracking
// - Alerting by error_code
//
// Top-level fields are considered stable monitoring contract.
// Additional business-specific fields must go into Metadata.
//
// ⚠ DO NOT log sensitive information such as:
// - password
// - token
// - card number
// - full request body
//
// Example:
//
//	TransactionLog{
//	    Operation:     "payment_process",
//	    TransactionID: "TXN-20260213-0001",
//	    UserID:        "user_12345",
//	    Status:        "failed",
//	    DurationMs:    120,
//	    ErrorCode:     "PAYMENT_TIMEOUT",
//	    Metadata: map[string]interface{}{
//	        "feature":     "transfer",
//	        "product":     "bank_transfer",
//	        "provider":    "bca",
//	        "channel":     "mobile_app",
//	        "amount":      150000,
//	        "corr_id":     "CORR-123",
//	        "external_id": "EXT-987",
//	    },
//	}
type TransactionLog struct {
	Operation     string                 // e.g. "payment_process", "order_submit"
	TransactionID string                 // business transaction ID
	UserID        string                 // user identifier
	Status        string                 // "success" or "failed"
	DurationMs    int64                  // execution duration in milliseconds
	ErrorCode     string                 // optional classification (e.g. "PAYMENT_TIMEOUT")
	Metadata      map[string]interface{} // additional structured business info
}

func (z *zapLogger) LogTransaction(ctx context.Context, tx TransactionLog) {
	fields := []Field{
		{Key: "category", Value: "transaction"},
		{Key: "operation", Value: tx.Operation},
		{Key: "transaction_id", Value: tx.TransactionID},
		{Key: "user_id", Value: tx.UserID},
		{Key: "status", Value: tx.Status},
		{Key: "duration_ms", Value: tx.DurationMs},
	}

	if tx.ErrorCode != "" {
		fields = append(fields, Field{Key: "error_code", Value: tx.ErrorCode})
	}

	if tx.Metadata != nil {
		fields = append(fields, Field{Key: "metadata", Value: tx.Metadata})
	}

	z.Info(ctx, "transaction_log", fields...)
}
