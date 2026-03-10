package messaging

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"

	"github.com/segmentio/kafka-go"
	"github.com/yogayulanda/go-core/config"
	"github.com/yogayulanda/go-core/logger"
)

var ErrPublisherClosed = errors.New("publisher already closed")

type kafkaPublisher struct {
	writer  *kafka.Writer
	cfg     publisherConfig
	closed  atomic.Bool
	closeMu sync.Mutex
}

func NewKafkaPublisher(
	cfg config.KafkaConfig,
	opts ...PublisherOption,
) (Publisher, error) {

	writer := &kafka.Writer{
		Addr:         kafka.TCP(cfg.Brokers...),
		Balancer:     &kafka.LeastBytes{},
		Async:        false,
		RequiredAcks: kafka.RequireAll,
	}

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

		if dlqErr != nil && k.cfg.log != nil {
			k.cfg.log.Error(ctx, "dlq publish failed",
				logger.Field{Key: "topic", Value: dlqMsg.Topic},
				logger.Field{Key: "error", Value: dlqErr},
			)
		}
	}

	if err != nil && k.cfg.log != nil {
		k.cfg.log.Error(ctx, "kafka publish failed",
			logger.Field{Key: "topic", Value: msg.Topic},
			logger.Field{Key: "error", Value: err},
		)
		return err
	}

	if err == nil && k.cfg.log != nil && k.cfg.successLogging {
		k.cfg.log.Info(ctx, "kafka publish success",
			logger.Field{Key: "topic", Value: msg.Topic},
		)
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
