package messaging

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"time"

	"github.com/segmentio/kafka-go"
	"github.com/yogayulanda/go-core/config"
)

var ErrPublisherClosed = errors.New("publisher already closed")

type kafkaMessageWriter interface {
	WriteMessages(ctx context.Context, msgs ...kafka.Message) error
	Close() error
}

var newKafkaWriter = func(cfg config.KafkaConfig) kafkaMessageWriter {
	return &kafka.Writer{
		Addr:         kafka.TCP(cfg.Brokers...),
		Balancer:     &kafka.LeastBytes{},
		Async:        false,
		RequiredAcks: kafka.RequireAll,
	}
}

type kafkaPublisher struct {
	writer  kafkaMessageWriter
	cfg     publisherConfig
	closed  atomic.Bool
	closeMu sync.Mutex
}

func NewKafkaPublisher(
	cfg config.KafkaConfig,
	opts ...PublisherOption,
) (Publisher, error) {
	writer := newKafkaWriter(cfg)

	conf := publisherConfig{}

	for _, opt := range opts {
		opt(&conf)
	}

	return &kafkaPublisher{
		writer: writer,
		cfg:    conf,
	}, nil
}

func (k *kafkaPublisher) Publish(ctx context.Context, msg Message) error {
	if k.closed.Load() {
		return ErrPublisherClosed
	}
	startedAt := time.Now()

	kMsg := kafka.Message{
		Topic: msg.Topic,
		Key:   msg.Key,
		Value: msg.Payload,
	}

	for key, val := range msg.Headers {
		kMsg.Headers = append(kMsg.Headers, kafka.Header{
			Key:   key,
			Value: []byte(val),
		})
	}

	err := executeWithRetry(
		ctx,
		k.cfg.retryEnabled,
		k.cfg.maxRetries,
		k.cfg.retryDelay,
		func(execCtx context.Context) error {
			return k.writer.WriteMessages(execCtx, kMsg)
		},
	)

	// DLQ
	if err != nil && k.cfg.dlqEnabled {
		dlqMsg := kafka.Message{
			Topic: msg.Topic + ".dlq",
			Key:   msg.Key,
			Value: msg.Payload,
		}

		dlqErr := k.writer.WriteMessages(ctx, dlqMsg)
		dlqStatus := "dlq_success"
		errorCode := ""
		metadata := map[string]interface{}{
			"topic":         msg.Topic,
			"dlq_topic":     dlqMsg.Topic,
			"retry_enabled": k.cfg.retryEnabled,
			"dlq_enabled":   k.cfg.dlqEnabled,
		}
		if dlqErr != nil {
			dlqStatus = "dlq_failed"
			errorCode = "dlq_publish_failed"
			metadata["error"] = dlqErr.Error()
		}
		observeMessagePublish(k.cfg, msg.Topic, dlqStatus)
		logMessagePublish(ctx, k.cfg, msg.Topic, dlqStatus, errorCode, metadata, time.Since(startedAt))
	}

	if err != nil {
		observeMessagePublish(k.cfg, msg.Topic, "failed")
		logMessagePublish(ctx, k.cfg, msg.Topic, "failed", "publish_failed", map[string]interface{}{
			"topic":         msg.Topic,
			"retry_enabled": k.cfg.retryEnabled,
			"dlq_enabled":   k.cfg.dlqEnabled,
			"error":         err.Error(),
		}, time.Since(startedAt))
		return err
	}

	observeMessagePublish(k.cfg, msg.Topic, "success")
	if k.cfg.successLogging {
		logMessagePublish(ctx, k.cfg, msg.Topic, "success", "", map[string]interface{}{
			"topic":         msg.Topic,
			"retry_enabled": k.cfg.retryEnabled,
			"dlq_enabled":   k.cfg.dlqEnabled,
		}, time.Since(startedAt))
	}

	return err
}

func (k *kafkaPublisher) Close() error {

	if k.closed.Load() {
		return nil
	}

	k.closeMu.Lock()
	defer k.closeMu.Unlock()

	if k.closed.Load() {
		return nil
	}

	err := k.writer.Close()
	k.closed.Store(true)

	return err
}
