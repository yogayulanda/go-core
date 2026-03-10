package messaging

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/segmentio/kafka-go"
	"github.com/yogayulanda/go-core/config"
	"github.com/yogayulanda/go-core/logger"
)

type kafkaConsumer struct {
	reader  *kafka.Reader
	handler Handler
	cfg     consumerConfig

	wg sync.WaitGroup
}

func NewKafkaConsumer(
	cfg config.KafkaConfig,
	topic string,
	groupID string,
	handler Handler,
	opts ...ConsumerOption,
) (Consumer, error) {

	if topic == "" {
		return nil, errors.New("topic cannot be empty")
	}

	if groupID == "" {
		return nil, errors.New("groupID cannot be empty")
	}
	if handler == nil {
		return nil, errors.New("handler cannot be nil")
	}

	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:        cfg.Brokers,
		Topic:          topic,
		GroupID:        groupID,
		MinBytes:       1,
		MaxBytes:       10e6,
		CommitInterval: 0, // manual commit
	})

	conf := consumerConfig{
		workerCount: 1,
	}

	for _, opt := range opts {
		opt(&conf)
	}

	return &kafkaConsumer{
		reader:  reader,
		handler: handler,
		cfg:     conf,
	}, nil
}

func (k *kafkaConsumer) Start(ctx context.Context) error {

	msgChan := make(chan kafka.Message)

	// ==============================
	// Worker Pool
	// ==============================

	for i := 0; i < k.cfg.workerCount; i++ {

		k.wg.Add(1)

		go func() {
			defer k.wg.Done()

			for m := range msgChan {

				msg := Message{
					Topic:   m.Topic,
					Key:     m.Key,
					Payload: m.Value,
					Headers: map[string]string{},
				}

				for _, h := range m.Headers {
					msg.Headers[h.Key] = string(h.Value)
				}

				var err error
				shouldCommit := false

				err = executeWithRetry(
					ctx,
					k.cfg.retryEnabled,
					k.cfg.maxRetry,
					k.cfg.retryDelay,
					func(execCtx context.Context) error {
						return k.handler(execCtx, msg)
					},
				)
				if err != nil && ctx.Err() != nil {
					return
				}

				if err == nil {
					shouldCommit = true
				}

				// ==============================
				// DLQ (If Needed)
				// ==============================

				if err != nil && k.cfg.dlqEnabled && k.cfg.dlqPublisher != nil {

					dlqMsg := Message{
						Topic:   msg.Topic + ".dlq",
						Key:     msg.Key,
						Payload: msg.Payload,
						Headers: msg.Headers,
					}

					dlqErr := k.cfg.dlqPublisher.Publish(ctx, dlqMsg)

					if dlqErr != nil {

						if k.cfg.log != nil {
							k.cfg.log.Error(ctx, "dlq publish failed",
								logger.Field{Key: "topic", Value: dlqMsg.Topic},
								logger.Field{Key: "error", Value: dlqErr},
							)
						}

						// DO NOT COMMIT if DLQ fails
						continue
					}

					// Commit original message if DLQ publish succeeded.
					shouldCommit = true
				}

				// ==============================
				// Commit ONLY if success OR DLQ success
				// ==============================

				if shouldCommit {
					if commitErr := k.reader.CommitMessages(ctx, m); commitErr != nil {
						if k.cfg.log != nil {
							k.cfg.log.Error(ctx, "commit failed",
								logger.Field{Key: "error", Value: commitErr},
							)
						}
					}
				}

				// ==============================
				// Logging
				// ==============================

				if err != nil {
					if k.cfg.log != nil {
						k.cfg.log.Error(ctx, "kafka consume failed",
							logger.Field{Key: "topic", Value: msg.Topic},
							logger.Field{Key: "error", Value: err},
						)
					}
				} else if k.cfg.successLogging && k.cfg.log != nil {
					k.cfg.log.Info(ctx, "kafka consume success",
						logger.Field{Key: "topic", Value: msg.Topic},
					)
				}
			}
		}()
	}

	// ==============================
	// Fetch Loop (Hardened)
	// ==============================

	for {

		m, err := k.reader.FetchMessage(ctx)

		if err != nil {

			// If shutdown
			if ctx.Err() != nil {
				close(msgChan)
				k.wg.Wait()
				return nil
			}

			// Transient error → retry with backoff
			if k.cfg.log != nil {
				k.cfg.log.Error(ctx, "fetch error",
					logger.Field{Key: "error", Value: err},
				)
			}

			time.Sleep(2 * time.Second)
			continue
		}

		select {
		case msgChan <- m:
		case <-ctx.Done():
			close(msgChan)
			k.wg.Wait()
			return nil
		}
	}
}

func (k *kafkaConsumer) Close() error {
	return k.reader.Close()
}
