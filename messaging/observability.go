package messaging

import (
	"context"
	"time"

	"github.com/yogayulanda/go-core/logger"
)

func logMessagePublish(ctx context.Context, cfg publisherConfig, topic string, status string, errorCode string, metadata map[string]interface{}, duration time.Duration) {
	if cfg.log == nil {
		return
	}
	cfg.log.LogService(ctx, logger.ServiceLog{
		Operation:  "message_publish",
		Status:     status,
		DurationMs: duration.Milliseconds(),
		ErrorCode:  errorCode,
		Metadata:   metadata,
	})
}

func observeMessagePublish(cfg publisherConfig, topic string, status string) {
	if cfg.metrics == nil || cfg.serviceName == "" {
		return
	}
	cfg.metrics.MessagePublishTotal.WithLabelValues(cfg.serviceName, topic, status).Inc()
}

func logMessageConsume(ctx context.Context, cfg consumerConfig, status string, errorCode string, metadata map[string]interface{}, duration time.Duration) {
	if cfg.log == nil {
		return
	}
	cfg.log.LogService(ctx, logger.ServiceLog{
		Operation:  "message_consume",
		Status:     status,
		DurationMs: duration.Milliseconds(),
		ErrorCode:  errorCode,
		Metadata:   metadata,
	})
}

func observeMessageConsume(cfg consumerConfig, topic string, groupID string, status string) {
	if cfg.metrics == nil || cfg.serviceName == "" {
		return
	}
	cfg.metrics.MessageConsumeTotal.WithLabelValues(cfg.serviceName, topic, groupID, status).Inc()
}

func observeMessageProcessDuration(cfg consumerConfig, topic string, groupID string, duration time.Duration) {
	if cfg.metrics == nil || cfg.serviceName == "" {
		return
	}
	cfg.metrics.MessageProcessDuration.WithLabelValues(cfg.serviceName, topic, groupID).Observe(duration.Seconds())
}
