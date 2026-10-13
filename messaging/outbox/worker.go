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

	batchSize     int
	interval      time.Duration
	stopChan      chan struct{}
	driver        string
	stopOnce      sync.Once
	started       bool
	startMu       sync.Mutex
	metrics       *observability.Metrics
	serviceName   string
	maxAttempts   int
	leaseDuration time.Duration
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

func WithWorkerMaxAttempts(maxAttempts int) WorkerOption {
	return func(w *Worker) {
		if maxAttempts > 0 {
			w.maxAttempts = maxAttempts
		}
	}
}

func WithWorkerLeaseDuration(duration time.Duration) WorkerOption {
	return func(w *Worker) {
		if duration > 0 {
			w.leaseDuration = duration
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
		db:            db,
		publisher:     pub,
		log:           log,
		batchSize:     50,
		interval:      2 * time.Second,
		stopChan:      make(chan struct{}),
		maxAttempts:   5,
		leaseDuration: 30 * time.Second,
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
	claimStartedAt := time.Now()
	tx, err := w.db.BeginTx(ctx, &sql.TxOptions{
		Isolation: sql.LevelReadCommitted,
	})
	if err != nil {
		w.observeBatch(ctx, "failed", "begin_tx_failed", 0, publishedCount, startedAt, err)
		return err
	}

	leaseUntil := time.Now().Add(w.leaseDuration)
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
		eventID string
		topic   string
		key     []byte
		payload []byte
		headers map[string]string
		attempt int
	}

	var events []event

	for rows.Next() {

		var e event
		var headersRaw []byte

		if err := rows.Scan(
			&e.id,
			&e.eventID,
			&e.topic,
			&e.key,
			&e.payload,
			&headersRaw,
			&e.attempt,
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
	if err := rows.Err(); err != nil {
		_ = tx.Rollback()
		w.observeBatch(ctx, "failed", "rows_failed", len(events), 0, claimStartedAt, err)
		return err
	}
	if err := rows.Close(); err != nil {
		_ = tx.Rollback()
		w.observeBatch(ctx, "failed", "rows_close_failed", len(events), 0, claimStartedAt, err)
		return err
	}

	if len(events) == 0 {
		_ = tx.Rollback()
		w.observeBatch(ctx, "empty", "", 0, 0, startedAt, nil)
		return nil
	}

	var batchErr error
	for _, e := range events {
		claimQuery, claimArgs := buildClaimEventQuery(w.driver, e.id, leaseUntil)
		if _, err := tx.ExecContext(ctx, claimQuery, claimArgs...); err != nil {
			_ = tx.Rollback()
			w.observeBatch(ctx, "failed", "claim_update_failed", len(events), 0, claimStartedAt, err)
			return err
		}
	}
	if err := tx.Commit(); err != nil {
		w.observeBatch(ctx, "failed", "claim_commit_failed", len(events), 0, claimStartedAt, err)
		return err
	}

	for _, e := range events {
		publishErr := w.publisher.Publish(ctx, messaging.Message{
			Topic:   e.topic,
			Key:     e.key,
			Payload: e.payload,
			Headers: e.headers,
			EventID: e.eventID,
		})
		var updateQuery string
		var updateArgs []interface{}
		if publishErr == nil {
			publishedCount++
			updateQuery, updateArgs = buildMarkPublishedQuery(w.driver, e.id)
		} else {
			updateQuery, updateArgs = buildMarkFailedQuery(w.driver, e.id, e.attempt+1, w.maxAttempts, publishErr.Error(), time.Now().Add(retryBackoff(e.attempt+1)))
		}
		if _, err := w.db.ExecContext(ctx, updateQuery, updateArgs...); err != nil {
			joined := errors.Join(publishErr, fmt.Errorf("update outbox event %d: %w", e.id, err))
			w.observeBatch(ctx, "failed", "finalize_failed", len(events), publishedCount, startedAt, joined)
			return joined
		}
		if publishErr != nil {
			batchErr = errors.Join(batchErr, fmt.Errorf("publish outbox event %s: %w", e.eventID, publishErr))
		}
	}
	if batchErr != nil {
		w.observeBatch(ctx, "partial_failure", "publish_failed", len(events), publishedCount, startedAt, batchErr)
		return batchErr
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
	if w.metrics != nil && w.metrics.Enabled && w.serviceName != "" {
		w.metrics.OutboxBatchTotal.WithLabelValues(w.serviceName, status).Inc()
		w.metrics.OutboxBatchDuration.WithLabelValues(w.serviceName).Observe(duration.Seconds())
		w.metrics.OutboxBatchSize.WithLabelValues(w.serviceName).Observe(float64(batchSize))
	}
}

func buildSelectPendingQuery(driver string, batchSize int) (string, []interface{}) {
	switch normalizeDriver(driver) {
	case "sqlserver":
		return `SELECT TOP (@p1) candidate.id, candidate.event_id, candidate.topic, candidate.msg_key, candidate.payload, candidate.headers, candidate.attempt FROM outbox_events candidate WITH (UPDLOCK, READPAST, ROWLOCK) WHERE ((candidate.status = 'PENDING' AND candidate.available_at <= @p2) OR (candidate.status = 'PROCESSING' AND candidate.locked_until <= @p2)) AND NOT EXISTS (SELECT 1 FROM outbox_events older WITH (READPAST) WHERE older.msg_key = candidate.msg_key AND older.id < candidate.id AND older.status <> 'SENT') ORDER BY candidate.id`, []interface{}{batchSize, time.Now()}
	default:
		return `SELECT candidate.id, candidate.event_id, candidate.topic, candidate.msg_key, candidate.payload, candidate.headers, candidate.attempt FROM outbox_events AS candidate WHERE ((candidate.status = 'PENDING' AND candidate.available_at <= ?) OR (candidate.status = 'PROCESSING' AND candidate.locked_until <= ?)) AND NOT EXISTS (SELECT 1 FROM outbox_events AS older WHERE older.msg_key = candidate.msg_key AND older.id < candidate.id AND older.status <> 'SENT') ORDER BY candidate.id LIMIT ? FOR UPDATE SKIP LOCKED`, []interface{}{time.Now(), time.Now(), batchSize}
	}
}

func buildMarkPublishedQuery(driver string, id int64) (string, []interface{}) {
	switch normalizeDriver(driver) {
	case "sqlserver":
		return "UPDATE outbox_events SET status = 'SENT', sent_at = SYSDATETIME(), locked_until = NULL, last_error = NULL WHERE id = @p1", []interface{}{id}
	default:
		return "UPDATE outbox_events SET status = 'SENT', sent_at = CURRENT_TIMESTAMP, locked_until = NULL, last_error = NULL WHERE id = ?", []interface{}{id}
	}
}

func buildClaimEventQuery(driver string, id int64, leaseUntil time.Time) (string, []interface{}) {
	if normalizeDriver(driver) == "sqlserver" {
		return "UPDATE outbox_events SET status = 'PROCESSING', locked_until = @p1 WHERE id = @p2", []interface{}{leaseUntil, id}
	}
	return "UPDATE outbox_events SET status = 'PROCESSING', locked_until = ? WHERE id = ?", []interface{}{leaseUntil, id}
}

func buildMarkFailedQuery(driver string, id int64, attempt, maxAttempts int, lastError string, availableAt time.Time) (string, []interface{}) {
	status := "PENDING"
	if attempt >= maxAttempts {
		status = "DEAD"
	}
	if normalizeDriver(driver) == "sqlserver" {
		return "UPDATE outbox_events SET status = @p1, attempt = @p2, last_error = @p3, available_at = @p4, locked_until = NULL WHERE id = @p5", []interface{}{status, attempt, lastError, availableAt, id}
	}
	return "UPDATE outbox_events SET status = ?, attempt = ?, last_error = ?, available_at = ?, locked_until = NULL WHERE id = ?", []interface{}{status, attempt, lastError, availableAt, id}
}

func retryBackoff(attempt int) time.Duration {
	if attempt < 1 {
		attempt = 1
	}
	if attempt > 8 {
		attempt = 8
	}
	return time.Second * time.Duration(1<<uint(attempt-1))
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
	if normalizeDriver(w.driver) == "" {
		return fmt.Errorf("unsupported outbox database driver %q (supported: sqlserver, mysql)", w.driver)
	}
	return nil
}
