package messaging

import (
	"time"

	"github.com/yogayulanda/go-core/logger"
)

type publisherConfig struct {
	retryEnabled bool
	maxRetries   int
	retryDelay   time.Duration

	dlqEnabled bool

	log logger.Logger

	successLogging bool
}

type PublisherOption func(*publisherConfig)

func WithPublisherRetry(maxRetries int, delay time.Duration) PublisherOption {
	return func(c *publisherConfig) {
		c.retryEnabled = true
		if maxRetries < 0 {
			maxRetries = 0
		}
		c.maxRetries = maxRetries
		if delay < 0 {
			delay = 0
		}
		c.retryDelay = delay
	}
}

func WithPublisherDLQ() PublisherOption {
	return func(c *publisherConfig) {
		c.dlqEnabled = true
	}
}

func WithPublisherLogger(log logger.Logger) PublisherOption {
	return func(c *publisherConfig) {
		c.log = log
	}
}

func WithPublisherSuccessLog() PublisherOption {
	return func(c *publisherConfig) {
		c.successLogging = true
	}
}
