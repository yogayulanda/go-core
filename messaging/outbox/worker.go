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
	"github.com/yogayulanda/go-core/observability"
)

var ErrWorkerAlreadyStarted = errors.New("outbox worker already started")

type Worker struct {
	db        *sql.DB
	publisher messaging.Publisher
	log       logger.Logger

	batchSize   int
	interval    time.Duration
	stopChan    chan struct{}
	driver      string
	stopOnce    sync.Once
	started     bool
	startMu     sync.Mutex
	metrics     *observability.Metrics
	serviceName string
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

func WithWorkerMetrics(metrics *observability.Metrics, serviceName string) WorkerOption {
	return func(w *Worker) {
		w.metrics = metrics
		w.serviceName = serviceName
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
	if err := w.StartChecked(ctx); err != nil {
		if w != nil && w.log != nil {
			w.log.LogService(ctx, logger.ServiceLog{
				Operation: "outbox_worker",
				Status:    "failed",
				ErrorCode: "start_failed",
				Metadata: map[string]interface{}{
					"error": err.Error(),
				},
			})
		}
		return
	}
}

func (w *Worker) StartChecked(ctx context.Context) error {
	if err := w.Validate(); err != nil {
		return err
	}
	w.startMu.Lock()
	defer w.startMu.Unlock()
	if w.started {
		return ErrWorkerAlreadyStarted
	}
	w.started = true
	w.log.LogService(ctx, logger.ServiceLog{
		Operation: "outbox_worker",
		Status:    "started",
		Metadata: map[string]interface{}{
			"driver":     w.driver,
			"batch_size": w.batchSize,
			"interval":   w.interval.String(),
		},
	})
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
				_ = w.RunOnce(ctx)
			}
		}
	}()
	return nil
}

func (w *Worker) Stop(ctx context.Context) error {
	if w != nil && w.log != nil {
		w.log.LogService(ctx, logger.ServiceLog{
			Operation: "outbox_worker",
			Status:    "shutdown_requested",
			Metadata: map[string]interface{}{
				"driver": w.driver,
			},
		})
	}
	if !w.started {
		return nil
	}
	w.stopOnce.Do(func() {
		close(w.stopChan)
	})
	return nil
}

func (w *Worker) RunOnce(ctx context.Context) error {
	if err := w.Validate(); err != nil {
		return err
	}
	startedAt := time.Now()
	publishedCount := 0
	tx, err := w.db.BeginTx(ctx, &sql.TxOptions{
		Isolation: sql.LevelReadCommitted,
	})
	if err != nil {
		w.observeBatch(ctx, "failed", "begin_tx_failed", 0, publishedCount, startedAt, err)
		return err
	}

	selectQuery, selectArgs := buildSelectPendingQuery(w.driver, w.batchSize)
	rows, err := tx.QueryContext(ctx, selectQuery, selectArgs...)

	if err != nil {
		_ = tx.Rollback()
		w.observeBatch(ctx, "failed", "query_failed", 0, publishedCount, startedAt, err)
		return err
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
			w.observeBatch(ctx, "failed", "scan_failed", len(events), publishedCount, startedAt, err)
			return err
		}

		if len(headersRaw) > 0 {
			_ = json.Unmarshal(headersRaw, &e.headers)
		}

		events = append(events, e)
	}

	if len(events) == 0 {
		_ = tx.Rollback()
		w.observeBatch(ctx, "empty", "", 0, 0, startedAt, nil)
		return nil
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
			w.observeBatch(ctx, "failed", "publish_failed", len(events), publishedCount, startedAt, err)
			return err
		}
		publishedCount++

		updateQuery, updateArgs := buildMarkPublishedQuery(w.driver, e.id)
		_, err = tx.ExecContext(ctx, updateQuery, updateArgs...)

		if err != nil {
			_ = tx.Rollback()
			w.observeBatch(ctx, "failed", "mark_published_failed", len(events), publishedCount, startedAt, err)
			return err
		}
	}

	if err := tx.Commit(); err != nil {
		w.observeBatch(ctx, "failed", "commit_failed", len(events), publishedCount, startedAt, err)
		return err
	}
	w.observeBatch(ctx, "success", "", len(events), publishedCount, startedAt, nil)
	return nil
}

func (w *Worker) observeBatch(ctx context.Context, status string, errorCode string, batchSize int, publishedCount int, startedAt time.Time, err error) {
	duration := time.Since(startedAt)
	metadata := map[string]interface{}{
		"driver":          w.driver,
		"batch_size":      batchSize,
		"published_count": publishedCount,
	}
	if err != nil {
		metadata["error"] = err.Error()
	}
	if w.log != nil {
		w.log.LogService(ctx, logger.ServiceLog{
			Operation:  "outbox_batch",
			Status:     status,
			DurationMs: duration.Milliseconds(),
			ErrorCode:  errorCode,
			Metadata:   metadata,
		})
	}
	if w.metrics != nil && w.serviceName != "" {
		w.metrics.OutboxBatchTotal.WithLabelValues(w.serviceName, status).Inc()
		w.metrics.OutboxBatchDuration.WithLabelValues(w.serviceName).Observe(duration.Seconds())
		w.metrics.OutboxBatchSize.WithLabelValues(w.serviceName).Observe(float64(batchSize))
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
