package outbox

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/yogayulanda/go-core/logger"
	"github.com/yogayulanda/go-core/messaging"
)

type Worker struct {
	db        *sql.DB
	publisher messaging.Publisher
	log       logger.Logger

	batchSize int
	interval  time.Duration
	stopChan  chan struct{}
	driver    string
	stopOnce  sync.Once
	started   bool
}

type WorkerOption func(*Worker)

func WithWorkerDriver(driver string) WorkerOption {
	return func(w *Worker) {
		w.driver = normalizeDriver(driver)
	}
}

func WithWorkerBatchSize(batchSize int) WorkerOption {
	return func(w *Worker) {
		if batchSize > 0 {
			w.batchSize = batchSize
		}
	}
}

func WithWorkerInterval(interval time.Duration) WorkerOption {
	return func(w *Worker) {
		if interval > 0 {
			w.interval = interval
		}
	}
}

func NewWorker(
	db *sql.DB,
	pub messaging.Publisher,
	log logger.Logger,
) *Worker {
	return NewWorkerWithOptions(db, pub, log)
}

func NewWorkerWithOptions(
	db *sql.DB,
	pub messaging.Publisher,
	log logger.Logger,
	opts ...WorkerOption,
) *Worker {
	worker := &Worker{
		db:        db,
		publisher: pub,
		log:       log,
		batchSize: 50,
		interval:  2 * time.Second,
		stopChan:  make(chan struct{}),
		driver:    "mysql",
	}

	for _, opt := range opts {
		if opt != nil {
			opt(worker)
		}
	}

	worker.driver = normalizeDriver(worker.driver)
	return worker
}

func (w *Worker) Start(ctx context.Context) {
	if w.db == nil || w.publisher == nil || w.log == nil {
		return
	}
	w.started = true

	go func() {

		ticker := time.NewTicker(w.interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-w.stopChan:
				return
			case <-ticker.C:
				w.processBatch(ctx)
			}
		}
	}()
}

func (w *Worker) Stop(ctx context.Context) error {
	_ = ctx
	if !w.started {
		return nil
	}
	w.stopOnce.Do(func() {
		close(w.stopChan)
	})
	return nil
}

func (w *Worker) processBatch(ctx context.Context) {

	tx, err := w.db.BeginTx(ctx, &sql.TxOptions{
		Isolation: sql.LevelReadCommitted,
	})
	if err != nil {
		w.log.Error(ctx, "outbox begin tx failed",
			logger.Field{Key: "error", Value: err},
		)
		return
	}

	selectQuery, selectArgs := buildSelectPendingQuery(w.driver, w.batchSize)
	rows, err := tx.QueryContext(ctx, selectQuery, selectArgs...)

	if err != nil {
		_ = tx.Rollback()
		w.log.Error(ctx, "outbox query failed",
			logger.Field{Key: "error", Value: err},
		)
		return
	}

	defer rows.Close()

	type event struct {
		id      int64
		topic   string
		key     []byte
		payload []byte
		headers map[string]string
	}

	var events []event

	for rows.Next() {

		var e event
		var headersRaw []byte

		if err := rows.Scan(
			&e.id,
			&e.topic,
			&e.key,
			&e.payload,
			&headersRaw,
		); err != nil {
			_ = tx.Rollback()
			return
		}

		if len(headersRaw) > 0 {
			_ = json.Unmarshal(headersRaw, &e.headers)
		}

		events = append(events, e)
	}

	if len(events) == 0 {
		_ = tx.Rollback()
		return
	}

	for _, e := range events {

		err := w.publisher.Publish(ctx, messaging.Message{
			Topic:   e.topic,
			Key:     e.key,
			Payload: e.payload,
			Headers: e.headers,
		})

		if err != nil {
			_ = tx.Rollback()
			w.log.Error(ctx, "outbox publish failed",
				logger.Field{Key: "event_id", Value: e.id},
				logger.Field{Key: "error", Value: err},
			)
			return
		}

		updateQuery, updateArgs := buildMarkPublishedQuery(w.driver, e.id)
		_, err = tx.ExecContext(ctx, updateQuery, updateArgs...)

		if err != nil {
			_ = tx.Rollback()
			return
		}
	}

	if err := tx.Commit(); err != nil {
		w.log.Error(ctx, "outbox commit failed",
			logger.Field{Key: "error", Value: err},
		)
	}
}

func buildSelectPendingQuery(driver string, batchSize int) (string, []interface{}) {
	colKey := keyColumnByDriver(driver)

	switch normalizeDriver(driver) {
	case "postgres":
		return fmt.Sprintf(
			`SELECT id, topic, %s, payload, headers FROM outbox_events WHERE published_at IS NULL ORDER BY id FOR UPDATE SKIP LOCKED LIMIT $1`,
			colKey,
		), []interface{}{batchSize}
	case "sqlserver":
		return fmt.Sprintf(
			`SELECT TOP (@p1) id, topic, %s, payload, headers FROM outbox_events WITH (UPDLOCK, READPAST, ROWLOCK) WHERE published_at IS NULL ORDER BY id`,
			colKey,
		), []interface{}{batchSize}
	default:
		return fmt.Sprintf(
			`SELECT id, topic, %s, payload, headers FROM outbox_events WHERE published_at IS NULL ORDER BY id LIMIT ? FOR UPDATE SKIP LOCKED`,
			colKey,
		), []interface{}{batchSize}
	}
}

func buildMarkPublishedQuery(driver string, id int64) (string, []interface{}) {
	switch normalizeDriver(driver) {
	case "postgres":
		return "UPDATE outbox_events SET published_at = NOW() WHERE id = $1", []interface{}{id}
	case "sqlserver":
		return "UPDATE outbox_events SET published_at = SYSDATETIME() WHERE id = @p1", []interface{}{id}
	default:
		return "UPDATE outbox_events SET published_at = CURRENT_TIMESTAMP WHERE id = ?", []interface{}{id}
	}
}

func (w *Worker) Driver() string {
	return strings.TrimSpace(w.driver)
}

func (w *Worker) Validate() error {
	if w == nil {
		return errors.New("outbox worker is nil")
	}
	if w.db == nil {
		return errors.New("outbox worker db is nil")
	}
	if w.publisher == nil {
		return errors.New("outbox worker publisher is nil")
	}
	if w.log == nil {
		return errors.New("outbox worker logger is nil")
	}
	return nil
}
