package outbox

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/yogayulanda/go-core/logger"
	"github.com/yogayulanda/go-core/messaging"
	"github.com/yogayulanda/go-core/observability"
)

func TestBuildSelectPendingQuery_ByDriver(t *testing.T) {
	tests := []struct {
		name        string
		driver      string
		wantContain string
	}{
		{
			name:        "mysql",
			driver:      "mysql",
			wantContain: "LIMIT ? FOR UPDATE SKIP LOCKED",
		},
		{
			name:        "sqlserver",
			driver:      "sqlserver",
			wantContain: "SELECT TOP (@p1)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			query, args := buildSelectPendingQuery(tt.driver, 50)
			if !strings.Contains(query, tt.wantContain) {
				t.Fatalf("query does not contain expected fragment: %s", query)
			}
			wantArgs := 2
			if tt.driver == "mysql" {
				wantArgs = 3
			}
			if len(args) != wantArgs {
				t.Fatalf("expected %d args, got %d", wantArgs, len(args))
			}
		})
	}
}

func TestBuildMarkPublishedQuery_ByDriver(t *testing.T) {
	tests := []struct {
		name        string
		driver      string
		wantContain string
	}{
		{
			name:        "mysql",
			driver:      "mysql",
			wantContain: "CURRENT_TIMESTAMP",
		},
		{
			name:        "sqlserver",
			driver:      "sqlserver",
			wantContain: "SYSDATETIME()",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			query, args := buildMarkPublishedQuery(tt.driver, 10)
			if !strings.Contains(query, tt.wantContain) {
				t.Fatalf("query does not contain expected fragment: %s", query)
			}
			if len(args) != 1 {
				t.Fatalf("expected 1 arg, got %d", len(args))
			}
		})
	}
}

func TestNormalizeDriver_UnknownReturnsEmpty(t *testing.T) {
	got := normalizeDriver("unknown")
	if got != "" {
		t.Fatalf("expected empty unsupported driver, got %s", got)
	}
}

func TestNewWorkerWithOptions_ApplyDriverBatchInterval(t *testing.T) {
	w := NewWorkerWithOptions(
		&sql.DB{},
		nil,
		nil,
		WithWorkerDriver("sqlserver"),
		WithWorkerBatchSize(100),
		WithWorkerInterval(5*time.Second),
	)

	if w.Driver() != "sqlserver" {
		t.Fatalf("expected sqlserver driver, got %s", w.Driver())
	}
	if w.batchSize != 100 {
		t.Fatalf("expected batch size 100, got %d", w.batchSize)
	}
	if w.interval != 5*time.Second {
		t.Fatalf("expected interval 5s, got %v", w.interval)
	}
}

type stubPublisher struct{}

func (s stubPublisher) Publish(ctx context.Context, msg messaging.Message) error { return nil }
func (s stubPublisher) Close() error                                             { return nil }

type stubLogger struct{}

func (stubLogger) Info(ctx context.Context, msg string, fields ...logger.Field)  {}
func (stubLogger) Error(ctx context.Context, msg string, fields ...logger.Field) {}
func (stubLogger) Debug(ctx context.Context, msg string, fields ...logger.Field) {}
func (stubLogger) Warn(ctx context.Context, msg string, fields ...logger.Field)  {}
func (stubLogger) LogService(ctx context.Context, s logger.ServiceLog)           {}
func (stubLogger) LogDB(ctx context.Context, d logger.DBLog)                     {}
func (stubLogger) LogEvent(ctx context.Context, e logger.EventLog)               {}
func (stubLogger) LogTransaction(ctx context.Context, tx logger.TransactionLog)  {}
func (stubLogger) WithComponent(component string) logger.Logger                  { return stubLogger{} }

func TestWorkerValidate_MissingDeps_ReturnError(t *testing.T) {
	w := &Worker{}
	if err := w.Validate(); err == nil {
		t.Fatalf("expected validation error")
	}
}

func TestWorkerValidate_AllDepsPresent_ReturnNil(t *testing.T) {
	w := NewWorkerWithOptions(&sql.DB{}, stubPublisher{}, stubLogger{}, WithWorkerDriver("mysql"))
	if err := w.Validate(); err != nil {
		t.Fatalf("unexpected validation error: %v", err)
	}
}

func TestWorkerValidate_UnsupportedDriver_ReturnsError(t *testing.T) {
	w := NewWorkerWithOptions(&sql.DB{}, stubPublisher{}, stubLogger{}, WithWorkerDriver("postgres"))
	if err := w.Validate(); err == nil {
		t.Fatal("expected unsupported driver error")
	}
}

func TestWorkerStop_NotStarted_NoPanic(t *testing.T) {
	w := &Worker{
		stopChan: make(chan struct{}),
	}
	if err := w.Stop(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestWorkerStartChecked_DoubleStartProtected(t *testing.T) {
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock new failed: %v", err)
	}
	defer db.Close()

	w := NewWorkerWithOptions(db, stubPublisher{}, stubLogger{}, WithWorkerDriver("mysql"))
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := w.StartChecked(ctx); err != nil {
		t.Fatalf("unexpected start error: %v", err)
	}
	if err := w.StartChecked(ctx); !errors.Is(err, ErrWorkerAlreadyStarted) {
		t.Fatalf("expected ErrWorkerAlreadyStarted, got %v", err)
	}
}

func TestWorkerRunOnce_EmptyBatch_EmitsMetric(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock new failed: %v", err)
	}
	defer db.Close()

	mock.ExpectBegin()
	mock.ExpectQuery("SELECT (.+) FROM outbox_events").WillReturnRows(
		sqlmock.NewRows([]string{"id", "event_id", "topic", "msg_key", "payload", "headers", "attempt"}),
	)
	mock.ExpectRollback()

	metrics := observability.NewMetrics()
	w := NewWorkerWithOptions(
		db,
		stubPublisher{},
		stubLogger{},
		WithWorkerMetrics(metrics, "outbox-empty-test"),
		WithWorkerDriver("mysql"),
	)

	if err := w.RunOnce(context.Background()); err != nil {
		t.Fatalf("unexpected run error: %v", err)
	}
	if got := testutil.ToFloat64(metrics.OutboxBatchTotal.WithLabelValues("outbox-empty-test", "empty")); got != 1 {
		t.Fatalf("expected empty batch count 1, got %v", got)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations not met: %v", err)
	}
}

func TestWorkerRunOnce_SuccessBatch_EmitsMetric(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock new failed: %v", err)
	}
	defer db.Close()

	mock.ExpectBegin()
	mock.ExpectQuery("SELECT (.+) FROM outbox_events").WillReturnRows(
		sqlmock.NewRows([]string{"id", "event_id", "topic", "msg_key", "payload", "headers", "attempt"}).
			AddRow(int64(1), "event-1", "record.created", []byte("k"), []byte("payload"), []byte(`{"content-type":"application/json"}`), 0),
	)
	mock.ExpectExec("UPDATE outbox_events SET status = 'PROCESSING'").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
	mock.ExpectExec("UPDATE outbox_events SET status = 'SENT'").WillReturnResult(sqlmock.NewResult(0, 1))

	metrics := observability.NewMetrics()
	w := NewWorkerWithOptions(
		db,
		stubPublisher{},
		stubLogger{},
		WithWorkerMetrics(metrics, "outbox-success-test"),
		WithWorkerDriver("mysql"),
	)

	if err := w.RunOnce(context.Background()); err != nil {
		t.Fatalf("unexpected run error: %v", err)
	}
	if got := testutil.ToFloat64(metrics.OutboxBatchTotal.WithLabelValues("outbox-success-test", "success")); got != 1 {
		t.Fatalf("expected success batch count 1, got %v", got)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations not met: %v", err)
	}
}

type failingPublisher struct{}

func (failingPublisher) Publish(ctx context.Context, msg messaging.Message) error {
	return errors.New("publish failed")
}

func (failingPublisher) Close() error { return nil }

type perKeyPublisher struct {
	calls []string
}

func (p *perKeyPublisher) Publish(_ context.Context, message messaging.Message) error {
	p.calls = append(p.calls, message.EventID)
	if message.EventID == "event-fail" {
		return errors.New("broker unavailable")
	}
	return nil
}

func (*perKeyPublisher) Close() error { return nil }

func TestWorkerRunOnce_PublishFailureDoesNotBlockOtherKey(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT (.+) FROM outbox_events").WillReturnRows(
		sqlmock.NewRows([]string{"id", "event_id", "topic", "msg_key", "payload", "headers", "attempt"}).
			AddRow(int64(1), "event-fail", "audit", []byte("key-a"), []byte("secret-payload"), []byte(`{}`), 0).
			AddRow(int64(2), "event-ok", "audit", []byte("key-b"), []byte("other-payload"), []byte(`{}`), 0),
	)
	mock.ExpectExec("UPDATE outbox_events SET status = 'PROCESSING'").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("UPDATE outbox_events SET status = 'PROCESSING'").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
	mock.ExpectExec(`UPDATE outbox_events SET status = \?`).WithArgs("PENDING", 1, "broker unavailable", sqlmock.AnyArg(), int64(1)).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("UPDATE outbox_events SET status = 'SENT'").WithArgs(int64(2)).WillReturnResult(sqlmock.NewResult(0, 1))
	publisher := &perKeyPublisher{}
	worker := NewWorkerWithOptions(db, publisher, stubLogger{}, WithWorkerDriver("mysql"))
	if err := worker.RunOnce(context.Background()); err == nil {
		t.Fatal("expected a joined publish error")
	}
	if len(publisher.calls) != 2 || publisher.calls[0] != "event-fail" || publisher.calls[1] != "event-ok" {
		t.Fatalf("worker stopped after one key failed: calls=%v", publisher.calls)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestBuildMarkFailedQueryMovesEventToDeadAtAttemptLimit(t *testing.T) {
	query, args := buildMarkFailedQuery("mysql", 7, 3, 3, "failure", time.Now())
	if !strings.Contains(query, "status = ?") || args[0] != "DEAD" || args[1] != 3 {
		t.Fatalf("attempt limit did not make the event DEAD: query=%s args=%v", query, args)
	}
}

func TestWorkerRunOnce_PublishFailure_ReturnsError(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock new failed: %v", err)
	}
	defer db.Close()

	mock.ExpectBegin()
	mock.ExpectQuery("SELECT (.+) FROM outbox_events").WillReturnRows(
		sqlmock.NewRows([]string{"id", "event_id", "topic", "msg_key", "payload", "headers", "attempt"}).
			AddRow(int64(1), "event-1", "record.created", []byte("k"), []byte("payload"), []byte(`{}`), 0),
	)
	mock.ExpectExec("UPDATE outbox_events SET status = 'PROCESSING'").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
	mock.ExpectExec("UPDATE outbox_events SET status = ?").WithArgs("PENDING", 1, "publish failed", sqlmock.AnyArg(), int64(1)).WillReturnResult(sqlmock.NewResult(0, 1))

	metrics := observability.NewMetrics()
	w := NewWorkerWithOptions(
		db,
		failingPublisher{},
		stubLogger{},
		WithWorkerMetrics(metrics, "outbox-failure-test"),
		WithWorkerDriver("mysql"),
	)

	if err := w.RunOnce(context.Background()); err == nil {
		t.Fatalf("expected publish failure")
	}
	if got := testutil.ToFloat64(metrics.OutboxBatchTotal.WithLabelValues("outbox-failure-test", "partial_failure")); got != 1 {
		t.Fatalf("expected partial failure batch count 1, got %v", got)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations not met: %v", err)
	}
}

func TestWorkerRunOnce_CommitFailure_ReturnsError(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock new failed: %v", err)
	}
	defer db.Close()

	mock.ExpectBegin()
	mock.ExpectQuery("SELECT (.+) FROM outbox_events").WillReturnRows(
		sqlmock.NewRows([]string{"id", "event_id", "topic", "msg_key", "payload", "headers", "attempt"}).
			AddRow(int64(1), "event-1", "record.created", []byte("k"), []byte("payload"), []byte(`{}`), 0),
	)
	mock.ExpectExec("UPDATE outbox_events SET status = 'PROCESSING'").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit().WillReturnError(errors.New("commit failed"))

	metrics := observability.NewMetrics()
	w := NewWorkerWithOptions(
		db,
		stubPublisher{},
		stubLogger{},
		WithWorkerMetrics(metrics, "outbox-commit-test"),
		WithWorkerDriver("mysql"),
	)

	if err := w.RunOnce(context.Background()); err == nil {
		t.Fatalf("expected commit failure")
	}
	if got := testutil.ToFloat64(metrics.OutboxBatchTotal.WithLabelValues("outbox-commit-test", "failed")); got != 1 {
		t.Fatalf("expected failed batch count 1, got %v", got)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations not met: %v", err)
	}
}
