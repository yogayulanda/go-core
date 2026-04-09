package migration

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/yogayulanda/go-core/config"
	"github.com/yogayulanda/go-core/logger"
)

func TestAutoRunUpWithRunnerAndLogger_AutoRunDisabled_LogsSkipped(t *testing.T) {
	log := &captureMigrationLogger{}
	cfg := minimalAutoRunConfig()
	cfg.Migration.AutoRun = false

	if err := AutoRunUpWithRunnerAndLogger(cfg, &fakeRunner{}, log); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(log.serviceLogs) != 1 {
		t.Fatalf("expected 1 log, got %d", len(log.serviceLogs))
	}
	entry := log.serviceLogs[0]
	if entry.Operation != "migration_autorun" || entry.Status != "skipped" {
		t.Fatalf("unexpected log entry: %+v", entry)
	}
}

func TestAutoRunUpWithRunnerAndLogger_Success_LogsAutorunAndLock(t *testing.T) {
	restore := overrideMigrationDeps(t)
	defer restore()

	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("create sqlmock failed: %v", err)
	}
	defer db.Close()

	openSQLDBFn = func(_ *config.Config, _ string) (*sql.DB, config.DBConfig, error) {
		return db, config.DBConfig{Driver: "mysql"}, nil
	}
	ensureGooseVersionTableFn = func(_ string, _ *sql.DB) error { return nil }
	acquireMigrationLockFn = func(_ context.Context, _ *sql.DB, _ string, _ string, _ time.Duration) (releaseLockFunc, error) {
		return func(context.Context) error { return nil }, nil
	}

	log := &captureMigrationLogger{}
	cfg := minimalAutoRunConfig()
	cfg.Migration.LockEnabled = true

	if err := AutoRunUpWithRunnerAndLogger(cfg, &fakeRunner{}, log); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !log.contains("migration_lock", "success") {
		t.Fatalf("expected migration_lock success log")
	}
	if !log.contains("migration_autorun", "success") {
		t.Fatalf("expected migration_autorun success log")
	}
}

type captureMigrationLogger struct {
	serviceLogs []logger.ServiceLog
}

func (l *captureMigrationLogger) Info(context.Context, string, ...logger.Field)         {}
func (l *captureMigrationLogger) Error(context.Context, string, ...logger.Field)        {}
func (l *captureMigrationLogger) Debug(context.Context, string, ...logger.Field)        {}
func (l *captureMigrationLogger) Warn(context.Context, string, ...logger.Field)         {}
func (l *captureMigrationLogger) LogDB(context.Context, logger.DBLog)                   {}
func (l *captureMigrationLogger) LogEvent(context.Context, logger.EventLog)             {}
func (l *captureMigrationLogger) LogTransaction(context.Context, logger.TransactionLog) {}
func (l *captureMigrationLogger) WithComponent(string) logger.Logger                    { return l }
func (l *captureMigrationLogger) LogService(_ context.Context, s logger.ServiceLog) {
	l.serviceLogs = append(l.serviceLogs, s)
}

func (l *captureMigrationLogger) contains(operation string, status string) bool {
	for _, entry := range l.serviceLogs {
		if entry.Operation == operation && entry.Status == status {
			return true
		}
	}
	return false
}
