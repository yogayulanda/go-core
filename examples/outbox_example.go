package examples

import (
	"context"
	"database/sql"
	"time"

	"github.com/yogayulanda/go-core/dbtx"
	"github.com/yogayulanda/go-core/logger"
	"github.com/yogayulanda/go-core/messaging"
	"github.com/yogayulanda/go-core/messaging/outbox"
	"github.com/yogayulanda/go-core/observability"
)

// WriteRecordAndQueueEventExample shows the recommended pattern:
// persist domain data and outbox event in one SQL transaction.
func WriteRecordAndQueueEventExample(
	ctx context.Context,
	db *sql.DB,
	repo *RecordSQLRepository,
	pub *outbox.Publisher,
	in CreateRecordInput,
) error {
	return dbtx.WithTx(ctx, db, func(txCtx context.Context) error {
		id, err := repo.Create(txCtx, in)
		if err != nil {
			return err
		}

		tx, ok := dbtx.FromContext(txCtx)
		if !ok || tx == nil {
			return nil
		}

		return pub.PublishTx(txCtx, tx, messaging.Message{
			Topic:   "record.created",
			Key:     []byte(id),
			Payload: []byte(`{"event":"record.created"}`),
			Headers: map[string]string{"content-type": "application/json"},
		})
	})
}

// PublishAuditEventExample stores CloudEvents metadata and payload in the caller's transaction.
func PublishAuditEventExample(ctx context.Context, tx *sql.Tx, publisher *outbox.Publisher) error {
	return publisher.PublishAudit(ctx, tx, outbox.AuditEvent{
		Topic:       "account.changed.v1",
		SpecVersion: "1.0",
		ID:          "11111111-1111-4111-8111-111111111111",
		Type:        "account.changed.v1",
		Source:      "/accounts/example",
		Time:        time.Now().UTC(),
		Subject:     "account-example",
		Sequence:    "1",
		Data:        []byte(`{"state":"changed"}`),
	})
}

// StartOutboxWorkerExample shows explicit service-owned worker startup.
func StartOutboxWorkerExample(
	ctx context.Context,
	db *sql.DB,
	pub messaging.Publisher,
	log logger.Logger,
	metrics *observability.Metrics,
	serviceName string,
) error {
	worker := outbox.NewWorkerWithOptions(
		db,
		pub,
		log,
		outbox.WithWorkerDriver("mysql"),
		outbox.WithWorkerBatchSize(50),
		outbox.WithWorkerInterval(2*time.Second),
		outbox.WithWorkerMetrics(metrics, serviceName),
	)

	return worker.StartChecked(ctx)
}
