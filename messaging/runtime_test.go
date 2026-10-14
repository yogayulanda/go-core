package messaging

import (
	"context"
	"errors"
	"log"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/segmentio/kafka-go"
	"github.com/yogayulanda/go-core/config"
	"github.com/yogayulanda/go-core/logger"
	"github.com/yogayulanda/go-core/observability"
)

type testLogger struct {
	mu          sync.Mutex
	serviceLogs []logger.ServiceLog
}

func (l *testLogger) Info(ctx context.Context, msg string, fields ...logger.Field)  {}
func (l *testLogger) Error(ctx context.Context, msg string, fields ...logger.Field) {}
func (l *testLogger) Debug(ctx context.Context, msg string, fields ...logger.Field) {}
func (l *testLogger) Warn(ctx context.Context, msg string, fields ...logger.Field)  {}
func (l *testLogger) LogDB(ctx context.Context, d logger.DBLog)                     {}
func (l *testLogger) LogEvent(ctx context.Context, e logger.EventLog)               {}
func (l *testLogger) LogTransaction(ctx context.Context, tx logger.TransactionLog)  {}
func (l *testLogger) WithComponent(component string) logger.Logger                  { return l }

func (l *testLogger) LogService(ctx context.Context, s logger.ServiceLog) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.serviceLogs = append(l.serviceLogs, s)
}

func (l *testLogger) contains(operation string, status string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	for _, item := range l.serviceLogs {
		if item.Operation == operation && item.Status == status {
			return true
		}
	}
	return false
}

type stubKafkaWriter struct {
	writeFunc func(ctx context.Context, msgs ...kafka.Message) error
	closeFunc func() error
}

func (s stubKafkaWriter) WriteMessages(ctx context.Context, msgs ...kafka.Message) error {
	return s.writeFunc(ctx, msgs...)
}

func (s stubKafkaWriter) Close() error {
	if s.closeFunc != nil {
		return s.closeFunc()
	}
	return nil
}

type stubKafkaReader struct {
	fetchFunc  func(ctx context.Context) (kafka.Message, error)
	commitFunc func(ctx context.Context, msgs ...kafka.Message) error
	closeFunc  func() error
}

func (s stubKafkaReader) FetchMessage(ctx context.Context) (kafka.Message, error) {
	return s.fetchFunc(ctx)
}

func (s stubKafkaReader) CommitMessages(ctx context.Context, msgs ...kafka.Message) error {
	if s.commitFunc != nil {
		return s.commitFunc(ctx, msgs...)
	}
	return nil
}

func (s stubKafkaReader) Close() error {
	if s.closeFunc != nil {
		return s.closeFunc()
	}
	return nil
}

type stubPublisher struct{}

func (stubPublisher) Publish(ctx context.Context, msg Message) error { return nil }
func (stubPublisher) Close() error                                   { return nil }

func TestKafkaPublisher_ObservabilitySuccess(t *testing.T) {
	originalFactory := newKafkaWriter
	defer func() { newKafkaWriter = originalFactory }()

	newKafkaWriter = func(cfg config.KafkaConfig) kafkaMessageWriter {
		return stubKafkaWriter{
			writeFunc: func(ctx context.Context, msgs ...kafka.Message) error { return nil },
		}
	}

	log := &testLogger{}
	metrics := observability.NewMetrics(true)
	pub, err := NewKafkaPublisher(
		config.KafkaConfig{Brokers: []string{"127.0.0.1:9092"}},
		WithPublisherLogger(log),
		WithPublisherMetrics(metrics, "publisher-success-test"),
		WithPublisherSuccessLog(),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	err = pub.Publish(context.Background(), Message{
		Topic:   "record.created",
		Key:     []byte("k"),
		Payload: []byte("v"),
	})
	if err != nil {
		t.Fatalf("unexpected publish error: %v", err)
	}

	if got := testutil.ToFloat64(metrics.MessagePublishTotal.WithLabelValues("publisher-success-test", "record.created", "success")); got != 1 {
		t.Fatalf("expected success publish count 1, got %v", got)
	}
	if !log.contains("message_publish", "success") {
		t.Fatalf("expected message_publish success service log")
	}
}

func TestMessagingMetricRecordingDisabledByDefault(t *testing.T) {
	disabled := observability.NewMetrics(false)
	observeMessagePublish(publisherConfig{metrics: disabled, serviceName: "metrics-disabled-test"}, "topic", "success")
	if got := testutil.ToFloat64(disabled.MessagePublishTotal.WithLabelValues("metrics-disabled-test", "topic", "success")); got != 0 {
		t.Fatalf("disabled metrics recorded a publish: %v", got)
	}
	enabled := observability.NewMetrics(true)
	observeMessagePublish(publisherConfig{metrics: enabled, serviceName: "metrics-enabled-test"}, "topic", "success")
	if got := testutil.ToFloat64(enabled.MessagePublishTotal.WithLabelValues("metrics-enabled-test", "topic", "success")); got != 1 {
		t.Fatalf("enabled metrics did not record a publish: %v", got)
	}
}

func TestKafkaPublisher_ObservabilityFailureAndDLQ(t *testing.T) {
	tests := []struct {
		name          string
		dlqErr        error
		wantDLQStatus string
	}{
		{name: "dlq success", wantDLQStatus: "dlq_success"},
		{name: "dlq failed", dlqErr: errors.New("dlq failed"), wantDLQStatus: "dlq_failed"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			originalFactory := newKafkaWriter
			defer func() { newKafkaWriter = originalFactory }()

			call := 0
			dlqHeaders := make(map[string]string)
			newKafkaWriter = func(cfg config.KafkaConfig) kafkaMessageWriter {
				return stubKafkaWriter{
					writeFunc: func(ctx context.Context, msgs ...kafka.Message) error {
						call++
						if call == 1 {
							return errors.New("publish failed")
						}
						for _, header := range msgs[0].Headers {
							dlqHeaders[header.Key] = string(header.Value)
						}
						return tt.dlqErr
					},
				}
			}

			log := &testLogger{}
			metrics := observability.NewMetrics(true)
			serviceName := "publisher-failure-test-" + strings.ReplaceAll(tt.name, " ", "-")
			pub, err := NewKafkaPublisher(
				config.KafkaConfig{Brokers: []string{"127.0.0.1:9092"}},
				WithPublisherLogger(log),
				WithPublisherMetrics(metrics, serviceName),
				WithPublisherDLQ(),
			)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			err = pub.Publish(context.Background(), Message{
				Topic:   "record.failed",
				Key:     []byte("k"),
				Payload: []byte("v"),
				Headers: map[string]string{"original": "kept"},
			})
			if err == nil {
				t.Fatalf("expected publish error")
			}
			if dlqHeaders["original"] != "kept" || dlqHeaders["x-original-topic"] != "record.failed" || dlqHeaders["x-dlq-reason"] == "" {
				t.Fatalf("DLQ headers were not preserved/enriched: %#v", dlqHeaders)
			}

			if got := testutil.ToFloat64(metrics.MessagePublishTotal.WithLabelValues(serviceName, "record.failed", "failed")); got != 1 {
				t.Fatalf("expected failed publish count 1, got %v", got)
			}
			if got := testutil.ToFloat64(metrics.MessagePublishTotal.WithLabelValues(serviceName, "record.failed", tt.wantDLQStatus)); got != 1 {
				t.Fatalf("expected %s publish count 1, got %v", tt.wantDLQStatus, got)
			}
			if !log.contains("message_publish", "failed") {
				t.Fatalf("expected failure publish service log")
			}
		})
	}
}

func TestKafkaConsumer_ObservabilitySuccess(t *testing.T) {
	originalFactory := newKafkaReader
	defer func() { newKafkaReader = originalFactory }()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	fetchCount := 0
	newKafkaReader = func(cfg config.KafkaConfig, topic string, groupID string, _ *log.Logger, _ *kafka.Dialer) kafkaMessageReader {
		return stubKafkaReader{
			fetchFunc: func(ctx context.Context) (kafka.Message, error) {
				fetchCount++
				if fetchCount == 1 {
					return kafka.Message{Topic: topic, Value: []byte("payload")}, nil
				}
				<-ctx.Done()
				return kafka.Message{}, ctx.Err()
			},
		}
	}

	log := &testLogger{}
	metrics := observability.NewMetrics(true)
	consumer, err := NewKafkaConsumer(
		config.KafkaConfig{Brokers: []string{"127.0.0.1:9092"}},
		"record.created",
		"group-a",
		func(ctx context.Context, msg Message) error {
			cancel()
			return nil
		},
		WithConsumerLogger(log),
		WithConsumerMetrics(metrics, "consumer-success-test"),
		WithConsumerSuccessLog(),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if err := consumer.Start(ctx); err != nil {
		t.Fatalf("unexpected start error: %v", err)
	}

	if got := testutil.ToFloat64(metrics.MessageConsumeTotal.WithLabelValues("consumer-success-test", "record.created", "group-a", "success")); got != 1 {
		t.Fatalf("expected success consume count 1, got %v", got)
	}
	if !log.contains("message_consume", "success") {
		t.Fatalf("expected message_consume success service log")
	}
}

func TestKafkaConsumer_ProcessesAndCommitsPartitionOffsetsInOrder(t *testing.T) {
	originalFactory := newKafkaReader
	defer func() { newKafkaReader = originalFactory }()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var fetchMu sync.Mutex
	fetchIndex := 0
	var commitMu sync.Mutex
	var committed []int64
	newKafkaReader = func(_ config.KafkaConfig, topic, _ string, _ *log.Logger, _ *kafka.Dialer) kafkaMessageReader {
		return stubKafkaReader{
			fetchFunc: func(ctx context.Context) (kafka.Message, error) {
				fetchMu.Lock()
				defer fetchMu.Unlock()
				fetchIndex++
				if fetchIndex <= 2 {
					return kafka.Message{Topic: topic, Partition: 3, Offset: int64(fetchIndex), Value: []byte{byte(fetchIndex)}}, nil
				}
				<-ctx.Done()
				return kafka.Message{}, ctx.Err()
			},
			commitFunc: func(_ context.Context, messages ...kafka.Message) error {
				commitMu.Lock()
				committed = append(committed, messages[0].Offset)
				if len(committed) == 2 {
					cancel()
				}
				commitMu.Unlock()
				return nil
			},
		}
	}
	var active atomic.Int32
	var handlerMu sync.Mutex
	var handled []int
	consumer, err := NewKafkaConsumer(config.KafkaConfig{Brokers: []string{"127.0.0.1:9092"}}, "ordered", "ordered-group", func(_ context.Context, message Message) error {
		if active.Add(1) != 1 {
			t.Error("same partition messages were handled concurrently")
		}
		handlerMu.Lock()
		handled = append(handled, int(message.Payload[0]))
		handlerMu.Unlock()
		active.Add(-1)
		return nil
	}, WithConsumerConcurrency(4))
	if err != nil {
		t.Fatal(err)
	}
	if err := consumer.Start(ctx); err != nil {
		t.Fatal(err)
	}
	if len(handled) != 2 || handled[0] != 1 || handled[1] != 2 {
		t.Fatalf("partition handler order was not preserved: %v", handled)
	}
	if len(committed) != 2 || committed[0] != 1 || committed[1] != 2 {
		t.Fatalf("partition commit order was not preserved: %v", committed)
	}
}

func TestKafkaConsumer_ObservabilityCommitFailed(t *testing.T) {
	originalFactory := newKafkaReader
	defer func() { newKafkaReader = originalFactory }()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	fetchCount := 0
	newKafkaReader = func(cfg config.KafkaConfig, topic string, groupID string, _ *log.Logger, _ *kafka.Dialer) kafkaMessageReader {
		return stubKafkaReader{
			fetchFunc: func(ctx context.Context) (kafka.Message, error) {
				fetchCount++
				if fetchCount == 1 {
					return kafka.Message{Topic: topic, Value: []byte("payload")}, nil
				}
				<-ctx.Done()
				return kafka.Message{}, ctx.Err()
			},
			commitFunc: func(ctx context.Context, msgs ...kafka.Message) error {
				cancel()
				return errors.New("commit failed")
			},
		}
	}

	log := &testLogger{}
	metrics := observability.NewMetrics(true)
	consumer, err := NewKafkaConsumer(
		config.KafkaConfig{Brokers: []string{"127.0.0.1:9092"}},
		"record.failed",
		"group-b",
		func(ctx context.Context, msg Message) error { return nil },
		WithConsumerLogger(log),
		WithConsumerMetrics(metrics, "consumer-commit-test"),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if err := consumer.Start(ctx); err != nil {
		t.Fatalf("unexpected start error: %v", err)
	}

	if got := testutil.ToFloat64(metrics.MessageConsumeTotal.WithLabelValues("consumer-commit-test", "record.failed", "group-b", "commit_failed")); got != 1 {
		t.Fatalf("expected commit_failed count 1, got %v", got)
	}
	if !log.contains("message_consume", "failed") {
		t.Fatalf("expected message_consume failed service log for commit failure")
	}
}

func TestKafkaConsumer_ObservabilityDLQSuccessAndFetchFailed(t *testing.T) {
	originalFactory := newKafkaReader
	defer func() { newKafkaReader = originalFactory }()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	fetchCount := 0
	newKafkaReader = func(cfg config.KafkaConfig, topic string, groupID string, _ *log.Logger, _ *kafka.Dialer) kafkaMessageReader {
		return stubKafkaReader{
			fetchFunc: func(ctx context.Context) (kafka.Message, error) {
				fetchCount++
				switch fetchCount {
				case 1:
					return kafka.Message{Topic: topic, Value: []byte("payload")}, nil
				case 2:
					return kafka.Message{}, errors.New("fetch failed")
				default:
					cancel()
					return kafka.Message{}, ctx.Err()
				}
			},
		}
	}

	log := &testLogger{}
	metrics := observability.NewMetrics(true)
	consumer, err := NewKafkaConsumer(
		config.KafkaConfig{Brokers: []string{"127.0.0.1:9092"}},
		"record.dlq",
		"group-c",
		func(ctx context.Context, msg Message) error { return errors.New("handler failed") },
		WithConsumerLogger(log),
		WithConsumerMetrics(metrics, "consumer-dlq-test"),
		WithConsumerDLQ(stubPublisher{}),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if err := consumer.Start(ctx); err != nil {
		t.Fatalf("unexpected start error: %v", err)
	}

	if got := testutil.ToFloat64(metrics.MessageConsumeTotal.WithLabelValues("consumer-dlq-test", "record.dlq", "group-c", "dlq_success")); got != 1 {
		t.Fatalf("expected dlq_success count 1, got %v", got)
	}
	if got := testutil.ToFloat64(metrics.MessageConsumeTotal.WithLabelValues("consumer-dlq-test", "record.dlq", "group-c", "fetch_failed")); got != 1 {
		t.Fatalf("expected fetch_failed count 1, got %v", got)
	}
	if !log.contains("message_consume", "dlq_success") {
		t.Fatalf("expected message_consume dlq_success service log")
	}
}
