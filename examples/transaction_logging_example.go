package examples

import (
	"context"

	"github.com/yogayulanda/go-core/logger"
	"github.com/yogayulanda/go-core/observability"
)

// LogTransactionExample shows how a transaction-oriented service can emit
// the shared transaction observability contract.
func LogTransactionExample(ctx context.Context, log logger.Logger, transactionID string, userID string, status string) {
	if log == nil {
		return
	}
	ctx = observability.WithGTID(ctx, transactionID)
	_ = observability.SetUserID(ctx, userID)

	log.LogTransaction(ctx, logger.TransactionLog{
		Feature:      "payment",
		Product:      "bank_transfer",
		Status:       status,
		ResponseCode: "CORE-0000",
		Amount:       150000,
		DurationMs:   120,
	})
}
