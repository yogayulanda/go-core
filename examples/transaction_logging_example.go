package examples

import (
	"context"
	"time"

	"github.com/yogayulanda/go-core/logger"
)

// LogTransactionExample shows how a transaction-oriented service can emit
// the shared transaction observability contract.
func LogTransactionExample(ctx context.Context, log logger.Logger, transactionID string, userID string, status string) {
	if log == nil {
		return
	}

	log.LogTransaction(ctx, logger.TransactionLog{
		Operation:     "payment_process",
		TransactionID: transactionID,
		UserID:        userID,
		Status:        status,
		DurationMs:    120,
		ErrorCode:     "",
		Metadata: map[string]interface{}{
			"channel":     "mobile_app",
			"occurred_at": time.Now().UTC().Format(time.RFC3339),
		},
	})
}
