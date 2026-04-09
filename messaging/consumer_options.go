package messaging

import (
	"time"

	"github.com/yogayulanda/go-core/logger"
	"github.com/yogayulanda/go-core/observability"
)

type consumerConfig struct {
	workerCount int

	retryEnabled bool
	maxRetry     int
	retryDelay   time.Duration

	dlqEnabled   bool
	dlqPublisher Publisher

	log         logger.Logger
	metrics     *observability.Metrics
	serviceName string

	successLogging bool
}

type ConsumerOption func(*consumerConfig)

func WithConsumerConcurrency(n int) ConsumerOption {
	return func(c *consumerConfig) {
		if n > 0 {
			c.workerCount = n
		}
	}
}

func WithConsumerRetry(max int, delay time.Duration) ConsumerOption {
	return func(c *consumerConfig) {
		c.retryEnabled = true
		if max < 0 {
			max = 0
		}
		c.maxRetry = max
		if delay < 0 {
			delay = 0
		}
		c.retryDelay = delay
	}
}

func WithConsumerDLQ(pub Publisher) ConsumerOption {
	return func(c *consumerConfig) {
		c.dlqEnabled = true
		c.dlqPublisher = pub
	}
}

func WithConsumerLogger(log logger.Logger) ConsumerOption {
	return func(c *consumerConfig) {
		c.log = log
	}
}

func WithConsumerMetrics(metrics *observability.Metrics, serviceName string) ConsumerOption {
	return func(c *consumerConfig) {
		c.metrics = metrics
		c.serviceName = serviceName
	}
}

func WithConsumerSuccessLog() ConsumerOption {
	return func(c *consumerConfig) {
		c.successLogging = true
	}
}
