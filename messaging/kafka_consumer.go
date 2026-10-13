package messaging

import (
	"context"
	"errors"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/segmentio/kafka-go"
	"github.com/yogayulanda/go-core/config"
	corelogger "github.com/yogayulanda/go-core/logger"
	"github.com/yogayulanda/go-core/observability"
)

type kafkaMessageReader interface {
	FetchMessage(ctx context.Context) (kafka.Message, error)
	CommitMessages(ctx context.Context, msgs ...kafka.Message) error
	Close() error
}

func restoreIncomingIdentity(ctx context.Context, headers map[string]string) context.Context {
	ctx = observability.EnsureFieldBag(ctx)
	if requestID := headers["x-request-id"]; requestID != "" {
		ctx = observability.WithRequestID(ctx, requestID)
	} else {
		ctx = observability.WithRequestID(ctx, observability.GenerateRequestID())
	}
	observability.RestoreIncomingIdentity(ctx, headers["x-user-id"], propagatedLabelsFromHeaders(headers))
	return withCloudEventAttributes(ctx, headers)
}

func propagatedLabelsFromHeaders(headers map[string]string) map[string]string {
	labels := make(map[string]string)
	for _, name := range observability.PropagatedLabels() {
		if value := headers[observability.LabelMetadataPrefix+name]; value != "" {
			labels[name] = value
		}
	}
	return labels
}

func withCloudEventAttributes(ctx context.Context, headers map[string]string) context.Context {
	if headers["ce_specversion"] == "" && headers["ce_id"] == "" && headers["ce_type"] == "" {
		return ctx
	}
	eventTime, _ := time.Parse(time.RFC3339Nano, headers["ce_time"])
	attributes := CloudEventAttributes{
		SpecVersion: headers["ce_specversion"],
		ID:          headers["ce_id"],
		Type:        headers["ce_type"],
		Source:      headers["ce_source"],
		Time:        eventTime,
		Subject:     headers["ce_subject"],
		Sequence:    strings.TrimSpace(headers["ce_sequence"]),
	}
	return context.WithValue(ctx, cloudEventContextKey{}, attributes)
}

var newKafkaReader = func(cfg config.KafkaConfig, topic string, groupID string, errorLogger *log.Logger, dialer *kafka.Dialer) kafkaMessageReader {
	return kafka.NewReader(kafka.ReaderConfig{
		Brokers:        cfg.Brokers,
		Topic:          topic,
		GroupID:        groupID,
		MinBytes:       1,
		MaxBytes:       10e6,
		CommitInterval: 0, // manual commit
		ErrorLogger:    errorLogger,
		Dialer:         dialer,
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

	conf := consumerConfig{
		workerCount: 1,
	}

	for _, opt := range opts {
		opt(&conf)
	}
	var errorLogger *log.Logger
	if conf.log != nil {
		errorLogger = corelogger.NewKafkaErrorLogger(conf.log)
	}
	dialer, err := buildKafkaDialer(cfg)
	if err != nil {
		return nil, err
	}
	reader := newKafkaReader(cfg, topic, groupID, errorLogger, dialer)

	return &kafkaConsumer{
		reader:  reader,
		handler: handler,
		cfg:     conf,
		topic:   topic,
		groupID: groupID,
	}, nil
}

func (k *kafkaConsumer) Start(ctx context.Context) error {
	workerCount := k.cfg.workerCount
	if workerCount < 1 {
		workerCount = 1
	}
	messageQueues := make([]chan kafka.Message, workerCount)

	// ==============================
	// Worker Pool
	// ==============================

	for i := 0; i < workerCount; i++ {
		messageQueues[i] = make(chan kafka.Message, 100)
		messages := messageQueues[i]
		k.wg.Add(1)

		go func() {
			defer k.wg.Done()
			partitionBlocked := false

			for m := range messages {
				if partitionBlocked {
					continue
				}
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
				ctx := restoreIncomingIdentity(ctx, msg.Headers)

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
					dlqMsg.Headers = cloneMessageHeaders(msg.Headers)
					dlqMsg.Headers["x-dlq-reason"] = err.Error()
					dlqMsg.Headers["x-original-topic"] = msg.Topic

					dlqErr := k.cfg.dlqPublisher.Publish(ctx, dlqMsg)

					if dlqErr != nil {
						partitionBlocked = true
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
				if err != nil && !shouldCommit {
					partitionBlocked = true
				}

				// ==============================
				// Commit ONLY if success OR DLQ success
				// ==============================

				if shouldCommit {
					if commitErr := k.reader.CommitMessages(ctx, m); commitErr != nil {
						partitionBlocked = true
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
				for _, queue := range messageQueues {
					close(queue)
				}
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
		case messageQueues[partitionWorker(m.Partition, workerCount)] <- m:
		case <-ctx.Done():
			for _, queue := range messageQueues {
				close(queue)
			}
			k.wg.Wait()
			return nil
		}
	}
}

func cloneMessageHeaders(headers map[string]string) map[string]string {
	copyOfHeaders := make(map[string]string, len(headers))
	for key, value := range headers {
		copyOfHeaders[key] = value
	}
	return copyOfHeaders
}

func partitionWorker(partition, workerCount int) int {
	if partition < 0 {
		partition = -partition
	}
	return partition % workerCount
}

func (k *kafkaConsumer) Close() error {
	return k.reader.Close()
}
