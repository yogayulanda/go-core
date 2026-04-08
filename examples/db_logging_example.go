package examples

import (
	"context"

	"github.com/yogayulanda/go-core/logger"
)

// LogDBExample shows how a service can emit the standard
// structured database log.
func LogDBExample(ctx context.Context, log logger.Logger, dbName string, query string, status string, durationMs int64) {
	if log == nil {
		return
	}

	log.LogDB(ctx, logger.DBLog{
		Operation:  "query",
		Query:      query,
		DBName:     dbName,
		Status:     status,
		DurationMs: durationMs,
		Metadata: map[string]interface{}{
			"component": "repository",
		},
	})
}
