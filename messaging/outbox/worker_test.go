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
			name:        "postgres",
			driver:      "postgres",
			wantContain: "FOR UPDATE SKIP LOCKED LIMIT $1",
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
			if len(args) != 1 {
				t.Fatalf("expected 1 arg, got %d", len(args))
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
			name:        "postgres",
			driver:      "postgres",
			wantContain: "NOW()",
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

func TestNormalizeDriver_UnknownFallbackToMySQL(t *testing.T) {
	got := normalizeDriver("unknown")
	if got != "mysql" {
		t.Fatalf("expected mysql fallback, got %s", got)
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
	w := NewWorkerWithOptions(&sql.DB{}, stubPublisher{}, stubLogger{})
	if err := w.Validate(); err != nil {
		t.Fatalf("unexpected validation error: %v", err)
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

	w := NewWorkerWithOptions(db, stubPublisher{}, stubLogger{})
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
		sqlmock.NewRows([]string{"id", "topic", "key", "payload", "headers"}),
	)
	mock.ExpectRollback()

	metrics := observability.NewMetrics()
	w := NewWorkerWithOptions(
		db,
		stubPublisher{},
		stubLogger{},
		WithWorkerMetrics(metrics, "outbox-empty-test"),
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
		sqlmock.NewRows([]string{"id", "topic", "key", "payload", "headers"}).
			AddRow(int64(1), "record.created", []byte("k"), []byte("payload"), []byte(`{"content-type":"application/json"}`)),
	)
	mock.ExpectExec("UPDATE outbox_events SET published_at").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	metrics := observability.NewMetrics()
	w := NewWorkerWithOptions(
		db,
		stubPublisher{},
		stubLogger{},
		WithWorkerMetrics(metrics, "outbox-success-test"),
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

func TestWorkerRunOnce_PublishFailure_ReturnsError(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock new failed: %v", err)
	}
	defer db.Close()

	mock.ExpectBegin()
	mock.ExpectQuery("SELECT (.+) FROM outbox_events").WillReturnRows(
		sqlmock.NewRows([]string{"id", "topic", "key", "payload", "headers"}).
			AddRow(int64(1), "record.created", []byte("k"), []byte("payload"), []byte(`{}`)),
	)
	mock.ExpectRollback()

	metrics := observability.NewMetrics()
	w := NewWorkerWithOptions(
		db,
		failingPublisher{},
		stubLogger{},
		WithWorkerMetrics(metrics, "outbox-failure-test"),
	)

	if err := w.RunOnce(context.Background()); err == nil {
		t.Fatalf("expected publish failure")
	}
	if got := testutil.ToFloat64(metrics.OutboxBatchTotal.WithLabelValues("outbox-failure-test", "failed")); got != 1 {
		t.Fatalf("expected failed batch count 1, got %v", got)
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
		sqlmock.NewRows([]string{"id", "topic", "key", "payload", "headers"}).
			AddRow(int64(1), "record.created", []byte("k"), []byte("payload"), []byte(`{}`)),
	)
	mock.ExpectExec("UPDATE outbox_events SET published_at").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit().WillReturnError(errors.New("commit failed"))

	metrics := observability.NewMetrics()
	w := NewWorkerWithOptions(
		db,
		stubPublisher{},
		stubLogger{},
		WithWorkerMetrics(metrics, "outbox-commit-test"),
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
