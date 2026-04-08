package examples

import (
	"context"

	"github.com/yogayulanda/go-core/logger"
)

// LogServiceExample shows how a service can emit the standard
// structured service-flow log.
func LogServiceExample(ctx context.Context, log logger.Logger, operation string, status string, durationMs int64) {
	if log == nil {
		return
	}

	log.LogService(ctx, logger.ServiceLog{
		Operation:  operation,
		Status:     status,
		DurationMs: durationMs,
		Metadata: map[string]interface{}{
			"layer": "service",
		},
	})
}
