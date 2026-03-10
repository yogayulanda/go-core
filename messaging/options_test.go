package messaging

import (
	"context"
	"testing"
	"time"

	"github.com/yogayulanda/go-core/config"
)

func TestConsumerOptions_ApplyConfig(t *testing.T) {
	cfg := consumerConfig{}
	WithConsumerConcurrency(3)(&cfg)
	WithConsumerRetry(2, time.Second)(&cfg)
	WithConsumerSuccessLog()(&cfg)

	if cfg.workerCount != 3 {
		t.Fatalf("expected workerCount 3, got %d", cfg.workerCount)
	}
	if !cfg.retryEnabled || cfg.maxRetry != 2 || cfg.retryDelay != time.Second {
		t.Fatalf("retry config not applied")
	}
	if !cfg.successLogging {
		t.Fatalf("expected success logging true")
	}
}

func TestConsumerOptions_NegativeValues_Normalized(t *testing.T) {
	cfg := consumerConfig{
		workerCount: 1,
	}

	WithConsumerConcurrency(0)(&cfg)
	WithConsumerRetry(-1, -1*time.Second)(&cfg)

	if cfg.workerCount != 1 {
		t.Fatalf("workerCount must keep previous positive value")
	}
	if cfg.maxRetry != 0 {
		t.Fatalf("maxRetry must be normalized to 0, got %d", cfg.maxRetry)
	}
	if cfg.retryDelay != 0 {
		t.Fatalf("retryDelay must be normalized to 0, got %v", cfg.retryDelay)
	}
}

func TestPublisherOptions_ApplyConfig(t *testing.T) {
	cfg := publisherConfig{}
	WithPublisherRetry(3, time.Second)(&cfg)
	WithPublisherDLQ()(&cfg)
	WithPublisherSuccessLog()(&cfg)

	if !cfg.retryEnabled || cfg.maxRetries != 3 || cfg.retryDelay != time.Second {
		t.Fatalf("retry config not applied")
	}
	if !cfg.dlqEnabled {
		t.Fatalf("expected dlq enabled")
	}
	if !cfg.successLogging {
		t.Fatalf("expected success logging true")
	}
}

func TestPublisherOptions_NegativeValues_Normalized(t *testing.T) {
	cfg := publisherConfig{}
	WithPublisherRetry(-2, -1*time.Second)(&cfg)

	if cfg.maxRetries != 0 {
		t.Fatalf("maxRetries must be normalized to 0, got %d", cfg.maxRetries)
	}
	if cfg.retryDelay != 0 {
		t.Fatalf("retryDelay must be normalized to 0, got %v", cfg.retryDelay)
	}
}

func TestNewKafkaConsumer_ValidationError(t *testing.T) {
	_, err := NewKafkaConsumer(config.KafkaConfig{Brokers: []string{"127.0.0.1:9092"}}, "", "group", func(ctx context.Context, msg Message) error {
		return nil
	})
	if err == nil {
		t.Fatalf("expected topic validation error")
	}

	_, err = NewKafkaConsumer(config.KafkaConfig{Brokers: []string{"127.0.0.1:9092"}}, "topic", "", func(ctx context.Context, msg Message) error {
		return nil
	})
	if err == nil {
		t.Fatalf("expected group validation error")
	}

	_, err = NewKafkaConsumer(
		config.KafkaConfig{Brokers: []string{"127.0.0.1:9092"}},
		"topic",
		"group",
		nil,
	)
	if err == nil {
		t.Fatalf("expected handler validation error")
	}
}

func TestKafkaPublisher_CloseIdempotentAndPublishAfterClose(t *testing.T) {
	pub, err := NewKafkaPublisher(config.KafkaConfig{
		Brokers: []string{"127.0.0.1:9092"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if err := pub.Close(); err != nil {
		t.Fatalf("unexpected close error: %v", err)
	}
	if err := pub.Close(); err != nil {
		t.Fatalf("second close must be idempotent: %v", err)
	}

	err = pub.Publish(context.Background(), Message{
		Topic:   "topic",
		Key:     []byte("k"),
		Payload: []byte("v"),
	})
	if err != ErrPublisherClosed {
		t.Fatalf("expected ErrPublisherClosed, got: %v", err)
	}
}
