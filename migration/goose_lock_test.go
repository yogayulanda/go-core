package migration

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/yogayulanda/go-core/config"
)

func TestDefaultMigrationLockKey_ServiceAndDB_ReturnExpectedKey(t *testing.T) {
	tests := []struct {
		name   string
		cfg    *config.Config
		dbName string
		want   string
	}{
		{
			name:   "nil config and empty db name",
			cfg:    nil,
			dbName: "",
			want:   "service:migration:default",
		},
		{
			name: "empty service name fallback",
			cfg: &config.Config{
				App: config.AppConfig{ServiceName: "   "},
			},
			dbName: "transaction",
			want:   "service:migration:transaction",
		},
		{
			name: "service and db name set",
			cfg: &config.Config{
				App: config.AppConfig{ServiceName: "transaction-history-service"},
			},
			dbName: "history",
			want:   "transaction-history-service:migration:history",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := defaultMigrationLockKey(tc.cfg, tc.dbName)
			if got != tc.want {
				t.Fatalf("expected %q, got %q", tc.want, got)
			}
		})
	}
}

func TestAcquireMigrationLock_UnsupportedDriver_ReturnNoopRelease(t *testing.T) {
	release, err := acquireMigrationLock(context.Background(), nil, "sqlite", "lock-key", 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if release == nil {
		t.Fatalf("expected release func")
	}
	if err := release(context.Background()); err != nil {
		t.Fatalf("unexpected release error: %v", err)
	}
}

func TestAcquireMigrationLock_MySQLZeroTimeout_UseDefaultTimeout(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("create sqlmock failed: %v", err)
	}
	defer db.Close()

	mock.ExpectQuery(`SELECT GET_LOCK\(\?, \?\)`).
		WithArgs("lock-key", 30).
		WillReturnRows(sqlmock.NewRows([]string{"got"}).AddRow(1))
	mock.ExpectQuery(`SELECT RELEASE_LOCK\(\?\)`).
		WithArgs("lock-key").
		WillReturnRows(sqlmock.NewRows([]string{"released"}).AddRow(1))

	release, err := acquireMigrationLock(context.Background(), db, "mysql", "lock-key", 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := release(context.Background()); err != nil {
		t.Fatalf("unexpected release error: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}

func TestAcquireMigrationLock_SQLServerDriver_RouteToSQLServer(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("create sqlmock failed: %v", err)
	}
	defer db.Close()

	mock.ExpectQuery(`sp_getapplock`).
		WithArgs("lock-key", 2000).
		WillReturnRows(sqlmock.NewRows([]string{"res"}).AddRow(0))
	mock.ExpectExec(`sp_releaseapplock`).
		WithArgs("lock-key").
		WillReturnResult(sqlmock.NewResult(0, 0))

	release, err := acquireMigrationLock(context.Background(), db, "sqlserver", "lock-key", 2*time.Second)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := release(context.Background()); err != nil {
		t.Fatalf("unexpected release error: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}

func TestAcquireMigrationLock_PostgresDriver_RouteToPostgres(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("create sqlmock failed: %v", err)
	}
	defer db.Close()

	mock.ExpectQuery(`SELECT pg_try_advisory_lock\(hashtext\(\$1\)\)`).
		WithArgs("lock-key").
		WillReturnRows(sqlmock.NewRows([]string{"locked"}).AddRow(true))
	mock.ExpectQuery(`SELECT pg_advisory_unlock\(hashtext\(\$1\)\)`).
		WithArgs("lock-key").
		WillReturnRows(sqlmock.NewRows([]string{"unlocked"}).AddRow(true))

	release, err := acquireMigrationLock(context.Background(), db, "postgres", "lock-key", time.Second)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := release(context.Background()); err != nil {
		t.Fatalf("unexpected release error: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}

func TestAcquireMySQLLock_LockNotAcquired_ReturnTimeout(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("create sqlmock failed: %v", err)
	}
	defer db.Close()

	mock.ExpectQuery(`SELECT GET_LOCK\(\?, \?\)`).
		WithArgs("lock-key", 1).
		WillReturnRows(sqlmock.NewRows([]string{"got"}).AddRow(0))

	_, err = acquireMySQLLock(context.Background(), db, "lock-key", 100*time.Millisecond)
	if err == nil {
		t.Fatalf("expected timeout error")
	}
	if !strings.Contains(err.Error(), "lock timeout") {
		t.Fatalf("expected timeout error message, got: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}

func TestAcquireSQLServerLock_NegativeResult_ReturnError(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("create sqlmock failed: %v", err)
	}
	defer db.Close()

	mock.ExpectQuery(`sp_getapplock`).
		WithArgs("lock-key", 2000).
		WillReturnRows(sqlmock.NewRows([]string{"res"}).AddRow(-1))

	_, err = acquireSQLServerLock(context.Background(), db, "lock-key", 2*time.Second)
	if err == nil {
		t.Fatalf("expected lock acquisition error")
	}
	if !strings.Contains(err.Error(), "code -1") {
		t.Fatalf("expected sqlserver code error, got: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}

func TestAcquireSQLServerLock_LockAcquired_ReleaseSucceeds(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("create sqlmock failed: %v", err)
	}
	defer db.Close()

	mock.ExpectQuery(`sp_getapplock`).
		WithArgs("lock-key", 2000).
		WillReturnRows(sqlmock.NewRows([]string{"res"}).AddRow(0))
	mock.ExpectExec(`sp_releaseapplock`).
		WithArgs("lock-key").
		WillReturnResult(sqlmock.NewResult(0, 0))

	release, err := acquireSQLServerLock(context.Background(), db, "lock-key", 2*time.Second)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := release(context.Background()); err != nil {
		t.Fatalf("unexpected release error: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}

func TestAcquireSQLServerLock_QueryError_ReturnError(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("create sqlmock failed: %v", err)
	}
	defer db.Close()

	mock.ExpectQuery(`sp_getapplock`).
		WithArgs("lock-key", 2000).
		WillReturnError(context.DeadlineExceeded)

	_, err = acquireSQLServerLock(context.Background(), db, "lock-key", 2*time.Second)
	if err == nil {
		t.Fatalf("expected error")
	}
	if !strings.Contains(err.Error(), "acquire sqlserver migration lock failed") {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}

func TestAcquirePostgresLock_LockAcquired_ReleaseSucceeds(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("create sqlmock failed: %v", err)
	}
	defer db.Close()

	mock.ExpectQuery(`SELECT pg_try_advisory_lock\(hashtext\(\$1\)\)`).
		WithArgs("lock-key").
		WillReturnRows(sqlmock.NewRows([]string{"locked"}).AddRow(true))
	mock.ExpectQuery(`SELECT pg_advisory_unlock\(hashtext\(\$1\)\)`).
		WithArgs("lock-key").
		WillReturnRows(sqlmock.NewRows([]string{"unlocked"}).AddRow(true))

	release, err := acquirePostgresLock(context.Background(), db, "lock-key", 100*time.Millisecond)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := release(context.Background()); err != nil {
		t.Fatalf("unexpected release error: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}

func TestAcquirePostgresLock_LockNotAcquired_ReturnTimeout(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("create sqlmock failed: %v", err)
	}
	defer db.Close()

	mock.ExpectQuery(`SELECT pg_try_advisory_lock\(hashtext\(\$1\)\)`).
		WithArgs("lock-key").
		WillReturnRows(sqlmock.NewRows([]string{"locked"}).AddRow(false))

	_, err = acquirePostgresLock(context.Background(), db, "lock-key", 0)
	if err == nil {
		t.Fatalf("expected timeout error")
	}
	if !strings.Contains(err.Error(), "lock timeout") {
		t.Fatalf("expected timeout error, got: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}

func TestAcquirePostgresLock_QueryError_ReturnError(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("create sqlmock failed: %v", err)
	}
	defer db.Close()

	mock.ExpectQuery(`SELECT pg_try_advisory_lock\(hashtext\(\$1\)\)`).
		WithArgs("lock-key").
		WillReturnError(context.DeadlineExceeded)

	_, err = acquirePostgresLock(context.Background(), db, "lock-key", time.Second)
	if err == nil {
		t.Fatalf("expected error")
	}
	if !strings.Contains(err.Error(), "acquire postgres migration lock failed") {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}

func TestAcquirePostgresLock_CanceledContext_ReturnContextError(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := acquirePostgresLock(ctx, nil, "lock-key", time.Second)
	if err == nil {
		t.Fatalf("expected canceled context error")
	}
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context canceled, got: %v", err)
	}
}

func TestAcquireMySQLLock_QueryError_ReturnError(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("create sqlmock failed: %v", err)
	}
	defer db.Close()

	mock.ExpectQuery(`SELECT GET_LOCK\(\?, \?\)`).
		WithArgs("lock-key", 1).
		WillReturnError(context.DeadlineExceeded)

	_, err = acquireMySQLLock(context.Background(), db, "lock-key", 100*time.Millisecond)
	if err == nil {
		t.Fatalf("expected error")
	}
	if !strings.Contains(err.Error(), "acquire mysql migration lock failed") {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}

func TestAcquireMySQLLock_InvalidResult_ReturnTimeoutError(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("create sqlmock failed: %v", err)
	}
	defer db.Close()

	mock.ExpectQuery(`SELECT GET_LOCK\(\?, \?\)`).
		WithArgs("lock-key", 1).
		WillReturnRows(sqlmock.NewRows([]string{"got"}).AddRow(nil))

	_, err = acquireMySQLLock(context.Background(), db, "lock-key", 100*time.Millisecond)
	if err == nil {
		t.Fatalf("expected error")
	}
	if !strings.Contains(err.Error(), "lock timeout") {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}
