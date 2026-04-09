package migration

import (
	"context"
	"time"

	"github.com/yogayulanda/go-core/logger"
)

func logMigration(
	ctx context.Context,
	log logger.Logger,
	operation string,
	status string,
	startedAt time.Time,
	errorCode string,
	metadata map[string]interface{},
) {
	if log == nil {
		return
	}

	log.LogService(ctx, logger.ServiceLog{
		Operation:  operation,
		Status:     status,
		DurationMs: time.Since(startedAt).Milliseconds(),
		ErrorCode:  errorCode,
		Metadata:   metadata,
	})
}
