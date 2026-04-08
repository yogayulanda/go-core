package outbox

import (
	"context"
	"database/sql"
	"strings"
	"testing"
	"time"

	"github.com/yogayulanda/go-core/logger"
	"github.com/yogayulanda/go-core/messaging"
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
func (stubLogger) LogService(ctx context.Context, s logger.ServiceLog)          {}
func (stubLogger) LogDB(ctx context.Context, d logger.DBLog)                    {}
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
