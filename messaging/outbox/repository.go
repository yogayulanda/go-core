package outbox

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/yogayulanda/go-core/logger"
	"github.com/yogayulanda/go-core/messaging"
	"github.com/yogayulanda/go-core/observability"
)

type Publisher struct {
	driver string
	log    logger.Logger
}

type PublisherOption func(*Publisher)

func WithPublisherDriver(driver string) PublisherOption {
	return func(p *Publisher) {
		p.driver = normalizeDriver(driver)
	}
}

func NewPublisher(opts ...PublisherOption) *Publisher {
	pub := &Publisher{}
	for _, opt := range opts {
		if opt != nil {
			opt(pub)
		}
	}
	pub.driver = normalizeDriver(pub.driver)
	return pub
}

func (p *Publisher) PublishTx(
	ctx context.Context,
	tx *sql.Tx,
	msg messaging.Message,
) error {
	if p == nil || normalizeDriver(p.driver) == "" {
		return fmt.Errorf("unsupported outbox database driver %q (supported: sqlserver, mysql)", p.driver)
	}

	if tx == nil {
		return fmt.Errorf("outbox transaction is nil")
	}
	msg = messagingIdentity(ctx, msg)
	if msg.EventID == "" {
		msg.EventID = uuid.NewString()
	}
	headersJSON, err := json.Marshal(msg.Headers)
	if err != nil {
		return fmt.Errorf("marshal outbox headers: %w", err)
	}
	query, args := buildInsertPendingQuery(
		p.driver,
		msg.EventID,
		msg.Topic,
		msg.Key,
		msg.Payload,
		headersJSON,
		"PENDING",
		time.Now(),
	)

	_, err = tx.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("insert outbox event: %w", err)
	}
	return nil
}

func messagingIdentity(ctx context.Context, msg messaging.Message) messaging.Message {
	if msg.Headers == nil {
		msg.Headers = make(map[string]string)
	}
	if msg.Headers["x-request-id"] == "" {
		msg.Headers["x-request-id"] = observability.GetRequestID(ctx)
	}
	if msg.Headers["x-user-id"] == "" {
		msg.Headers["x-user-id"] = observability.GetUserID(ctx)
	}
	labels := observability.Labels(ctx)
	for _, name := range observability.PropagatedLabels() {
		if msg.Headers[observability.LabelMetadataPrefix+name] == "" {
			msg.Headers[observability.LabelMetadataPrefix+name] = labels[name]
		}
	}
	return msg
}

func buildInsertPendingQuery(
	driver string,
	eventID string,
	topic string,
	key []byte,
	payload []byte,
	headersJSON []byte,
	status string,
	createdAt time.Time,
) (string, []interface{}) {
	switch normalizeDriver(driver) {
	case "sqlserver":
		return `INSERT INTO dbo.outbox_events (event_id, topic, msg_key, payload, headers, status, attempt, last_error, available_at, created_at) VALUES (@p1, @p2, @p3, @p4, @p5, @p6, 0, NULL, @p7, @p7)`, []interface{}{eventID, topic, key, payload, headersJSON, status, createdAt}
	default:
		return `INSERT INTO outbox_events (event_id, topic, msg_key, payload, headers, status, attempt, last_error, available_at, created_at) VALUES (?, ?, ?, ?, ?, ?, 0, NULL, ?, ?)`, []interface{}{eventID, topic, key, payload, headersJSON, status, createdAt, createdAt}
	}
}
