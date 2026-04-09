package cache

import (
	"context"
	"time"

	"github.com/yogayulanda/go-core/logger"
)

func logConnect(
	ctx context.Context,
	log logger.Logger,
	backend string,
	startedAt time.Time,
	status string,
	errorCode string,
	metadata map[string]interface{},
) {
	if log == nil {
		return
	}

	log.LogService(ctx, logger.ServiceLog{
		Operation:  "cache_connect",
		Status:     status,
		DurationMs: time.Since(startedAt).Milliseconds(),
		ErrorCode:  errorCode,
		Metadata: mergeMetadata(map[string]interface{}{
			"dependency_type": "cache",
			"backend":         backend,
		}, metadata),
	})
}

func mergeMetadata(base map[string]interface{}, extra map[string]interface{}) map[string]interface{} {
	if len(extra) == 0 {
		return base
	}

	out := make(map[string]interface{}, len(base)+len(extra))
	for k, v := range base {
		out[k] = v
	}
	for k, v := range extra {
		out[k] = v
	}
	return out
}
