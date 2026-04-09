package messaging

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/segmentio/kafka-go"
	"github.com/yogayulanda/go-core/config"
)

type kafkaMessageReader interface {
	FetchMessage(ctx context.Context) (kafka.Message, error)
	CommitMessages(ctx context.Context, msgs ...kafka.Message) error
	Close() error
}

var newKafkaReader = func(cfg config.KafkaConfig, topic string, groupID string) kafkaMessageReader {
	return kafka.NewReader(kafka.ReaderConfig{
		Brokers:        cfg.Brokers,
		Topic:          topic,
		GroupID:        groupID,
		MinBytes:       1,
		MaxBytes:       10e6,
		CommitInterval: 0, // manual commit
	})
}

type kafkaConsumer struct {
	reader  kafkaMessageReader
	handler Handler
	cfg     consumerConfig
	topic   string
	groupID string

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

	reader := newKafkaReader(cfg, topic, groupID)

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
		topic:   topic,
		groupID: groupID,
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
				processStartedAt := time.Now()

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
				finalStatus := ""
				finalErrorCode := ""
				finalMetadata := map[string]interface{}{
					"topic": msg.Topic,
					"group": k.groupID,
				}

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
					finalStatus = "success"
				}
				observeMessageProcessDuration(k.cfg, msg.Topic, k.groupID, time.Since(processStartedAt))

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
						finalStatus = "dlq_failed"
						finalErrorCode = "dlq_publish_failed"
						finalMetadata["error"] = dlqErr.Error()
						finalMetadata["dlq"] = true
						finalMetadata["headers"] = len(msg.Headers)
						observeMessageConsume(k.cfg, msg.Topic, k.groupID, finalStatus)
						logMessageConsume(ctx, k.cfg, "failed", finalErrorCode, finalMetadata, time.Since(processStartedAt))

						// DO NOT COMMIT if DLQ fails
						continue
					}

					// Commit original message if DLQ publish succeeded.
					finalStatus = "dlq_success"
					finalMetadata["dlq"] = true
					err = nil
					shouldCommit = true
				}

				// ==============================
				// Commit ONLY if success OR DLQ success
				// ==============================

				if shouldCommit {
					if commitErr := k.reader.CommitMessages(ctx, m); commitErr != nil {
						observeMessageConsume(k.cfg, msg.Topic, k.groupID, "commit_failed")
						logMessageConsume(ctx, k.cfg, "failed", "commit_failed", map[string]interface{}{
							"topic": msg.Topic,
							"group": k.groupID,
							"error": commitErr.Error(),
						}, time.Since(processStartedAt))
						continue
					}
				}

				// ==============================
				// Logging
				// ==============================

				if err != nil {
					finalStatus = "failed"
					finalErrorCode = "consume_failed"
					finalMetadata["error"] = err.Error()
				}

				if finalStatus == "" {
					finalStatus = "success"
				}
				observeMessageConsume(k.cfg, msg.Topic, k.groupID, finalStatus)
				if finalStatus != "success" || k.cfg.successLogging {
					logMessageConsume(ctx, k.cfg, finalStatus, finalErrorCode, finalMetadata, time.Since(processStartedAt))
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
			observeMessageConsume(k.cfg, k.topic, k.groupID, "fetch_failed")
			logMessageConsume(ctx, k.cfg, "failed", "fetch_failed", map[string]interface{}{
				"topic": k.topic,
				"group": k.groupID,
				"error": err.Error(),
			}, 0)

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
