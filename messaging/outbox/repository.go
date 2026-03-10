package outbox

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/yogayulanda/go-core/messaging"
)

type Publisher struct {
	driver string
}

type PublisherOption func(*Publisher)

func WithPublisherDriver(driver string) PublisherOption {
	return func(p *Publisher) {
		p.driver = normalizeDriver(driver)
	}
}

func NewPublisher(opts ...PublisherOption) *Publisher {
	pub := &Publisher{
		driver: "mysql",
	}
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

	headersJSON, _ := json.Marshal(msg.Headers)
	query, args := buildInsertPendingQuery(
		p.driver,
		uuid.NewString(),
		msg.Topic,
		msg.Key,
		msg.Payload,
		headersJSON,
		"PENDING",
		time.Now(),
	)

	_, err := tx.ExecContext(ctx, query, args...)

	return err
}

func buildInsertPendingQuery(
	driver string,
	id string,
	topic string,
	key []byte,
	payload []byte,
	headersJSON []byte,
	status string,
	createdAt time.Time,
) (string, []interface{}) {
	colKey := keyColumnByDriver(driver)

	switch normalizeDriver(driver) {
	case "postgres":
		return fmt.Sprintf(
			`INSERT INTO outbox_events (id, topic, %s, payload, headers, status, created_at) VALUES ($1, $2, $3, $4, $5, $6, $7)`,
			colKey,
		), []interface{}{id, topic, key, payload, headersJSON, status, createdAt}
	case "sqlserver":
		return fmt.Sprintf(
			`INSERT INTO outbox_events (id, topic, %s, payload, headers, status, created_at) VALUES (@p1, @p2, @p3, @p4, @p5, @p6, @p7)`,
			colKey,
		), []interface{}{id, topic, key, payload, headersJSON, status, createdAt}
	default:
		return fmt.Sprintf(
			`INSERT INTO outbox_events (id, topic, %s, payload, headers, status, created_at) VALUES (?, ?, ?, ?, ?, ?, ?)`,
			colKey,
		), []interface{}{id, topic, key, payload, headersJSON, status, createdAt}
	}
}
