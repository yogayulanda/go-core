package migration

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/yogayulanda/go-core/config"
)

type fakeRunner struct {
	called bool
	err    error
}

func (r *fakeRunner) Run(_ *sql.DB, _ string, _ string, _ string) error {
	r.called = true
	return r.err
}

func TestAutoRunUpWithRunner_AutoRunDisabled_SkipOpenAndRunner(t *testing.T) {
	restore := overrideMigrationDeps(t)
	defer restore()

	openCalled := false
	openSQLDBFn = func(_ *config.Config, _ string) (*sql.DB, config.DBConfig, error) {
		openCalled = true
		return nil, config.DBConfig{}, nil
	}

	runner := &fakeRunner{}
	cfg := minimalAutoRunConfig()
	cfg.Migration.AutoRun = false

	if err := AutoRunUpWithRunner(cfg, runner); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if openCalled {
		t.Fatalf("OpenSQLDB must not be called when auto-run is disabled")
	}
	if runner.called {
		t.Fatalf("runner must not be called when auto-run is disabled")
	}
}

func TestAutoRunUpWithRunner_LockEnabled_AcquireAndReleaseLock(t *testing.T) {
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

	lockAcquireCalled := false
	lockReleaseCalled := false
	acquireMigrationLockFn = func(
		_ context.Context,
		_ *sql.DB,
		_ string,
		_ string,
		_ time.Duration,
	) (releaseLockFunc, error) {
		lockAcquireCalled = true
		return func(context.Context) error {
			lockReleaseCalled = true
			return nil
		}, nil
	}

	runner := &fakeRunner{}
	cfg := minimalAutoRunConfig()
	cfg.Migration.LockEnabled = true

	if err := AutoRunUpWithRunner(cfg, runner); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !lockAcquireCalled {
		t.Fatalf("expected lock to be acquired")
	}
	if !lockReleaseCalled {
		t.Fatalf("expected lock release to be called")
	}
	if !runner.called {
		t.Fatalf("runner must be called")
	}
}

func TestAutoRunUpWithRunner_LockDisabled_SkipLock(t *testing.T) {
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

	lockAcquireCalled := false
	acquireMigrationLockFn = func(
		_ context.Context,
		_ *sql.DB,
		_ string,
		_ string,
		_ time.Duration,
	) (releaseLockFunc, error) {
		lockAcquireCalled = true
		return func(context.Context) error { return nil }, nil
	}

	runner := &fakeRunner{}
	cfg := minimalAutoRunConfig()
	cfg.Migration.LockEnabled = false

	if err := AutoRunUpWithRunner(cfg, runner); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if lockAcquireCalled {
		t.Fatalf("lock must not be acquired when lock is disabled")
	}
	if !runner.called {
		t.Fatalf("runner must be called")
	}
}

func TestAutoRunUpWithRunner_LockAcquireError_StopBeforeRunner(t *testing.T) {
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

	acquireMigrationLockFn = func(
		_ context.Context,
		_ *sql.DB,
		_ string,
		_ string,
		_ time.Duration,
	) (releaseLockFunc, error) {
		return nil, errors.New("lock failed")
	}

	runner := &fakeRunner{}
	cfg := minimalAutoRunConfig()
	cfg.Migration.LockEnabled = true

	err = AutoRunUpWithRunner(cfg, runner)
	if err == nil {
		t.Fatalf("expected error when lock acquisition fails")
	}
	if runner.called {
		t.Fatalf("runner must not be called when lock acquisition fails")
	}
}

func TestAutoRunUpWithRunner_NilRunner_ReturnError(t *testing.T) {
	cfg := minimalAutoRunConfig()
	err := AutoRunUpWithRunner(cfg, nil)
	if err == nil {
		t.Fatalf("expected error")
	}
	if !strings.Contains(err.Error(), "runner is nil") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAutoRunUpWithRunner_OpenDBError_ReturnError(t *testing.T) {
	restore := overrideMigrationDeps(t)
	defer restore()

	openSQLDBFn = func(_ *config.Config, _ string) (*sql.DB, config.DBConfig, error) {
		return nil, config.DBConfig{}, errors.New("open failed")
	}

	runner := &fakeRunner{}
	cfg := minimalAutoRunConfig()
	err := AutoRunUpWithRunner(cfg, runner)
	if err == nil {
		t.Fatalf("expected error")
	}
	if !strings.Contains(err.Error(), "open failed") {
		t.Fatalf("unexpected error: %v", err)
	}
	if runner.called {
		t.Fatalf("runner must not be called")
	}
}

func TestAutoRunUpWithRunner_EnsureVersionTableError_ReturnError(t *testing.T) {
	restore := overrideMigrationDeps(t)
	defer restore()

	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("create sqlmock failed: %v", err)
	}
	defer db.Close()

	openSQLDBFn = func(_ *config.Config, _ string) (*sql.DB, config.DBConfig, error) {
		return db, config.DBConfig{Driver: "sqlserver"}, nil
	}
	ensureGooseVersionTableFn = func(_ string, _ *sql.DB) error {
		return errors.New("ensure failed")
	}

	runner := &fakeRunner{}
	cfg := minimalAutoRunConfig()
	err = AutoRunUpWithRunner(cfg, runner)
	if err == nil {
		t.Fatalf("expected error")
	}
	if !strings.Contains(err.Error(), "ensure failed") {
		t.Fatalf("unexpected error: %v", err)
	}
	if runner.called {
		t.Fatalf("runner must not be called")
	}
}

func TestAutoRunUpWithRunner_RunnerError_PropagateError(t *testing.T) {
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

	runner := &fakeRunner{err: errors.New("runner failed")}
	cfg := minimalAutoRunConfig()
	cfg.Migration.LockEnabled = false

	err = AutoRunUpWithRunner(cfg, runner)
	if err == nil {
		t.Fatalf("expected runner error")
	}
	if !strings.Contains(err.Error(), "runner failed") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAutoRunUpWithRunner_EmptyLockKey_UseDefaultLockKey(t *testing.T) {
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

	var gotLockKey string
	acquireMigrationLockFn = func(
		_ context.Context,
		_ *sql.DB,
		_ string,
		lockKey string,
		_ time.Duration,
	) (releaseLockFunc, error) {
		gotLockKey = lockKey
		return func(context.Context) error { return nil }, nil
	}

	runner := &fakeRunner{}
	cfg := minimalAutoRunConfig()
	cfg.Migration.LockEnabled = true
	cfg.Migration.LockKey = ""
	cfg.Migration.DBName = "transaction"
	cfg.App.ServiceName = "ths"

	err = AutoRunUpWithRunner(cfg, runner)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotLockKey != "ths:migration:transaction" {
		t.Fatalf("expected default lock key, got: %q", gotLockKey)
	}
}

func TestAutoRunUpWithRunner_CustomLockKey_UseCustomLockKey(t *testing.T) {
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

	var gotLockKey string
	acquireMigrationLockFn = func(
		_ context.Context,
		_ *sql.DB,
		_ string,
		lockKey string,
		_ time.Duration,
	) (releaseLockFunc, error) {
		gotLockKey = lockKey
		return func(context.Context) error { return nil }, nil
	}

	runner := &fakeRunner{}
	cfg := minimalAutoRunConfig()
	cfg.Migration.LockEnabled = true
	cfg.Migration.LockKey = "custom-lock-key"

	err = AutoRunUpWithRunner(cfg, runner)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotLockKey != "custom-lock-key" {
		t.Fatalf("expected custom lock key, got: %q", gotLockKey)
	}
}

func minimalAutoRunConfig() *config.Config {
	return &config.Config{
		App: config.AppConfig{
			ServiceName: "transaction-history-service",
		},
		Databases: map[string]config.DBConfig{
			"transaction": {
				Driver: "mysql",
			},
		},
		Migration: config.MigrationConfig{
			AutoRun:     true,
			DBName:      "transaction",
			Dir:         "migrations/transaction",
			LockEnabled: true,
			LockTimeout: 30 * time.Second,
		},
	}
}

func overrideMigrationDeps(t *testing.T) func() {
	t.Helper()

	origOpen := openSQLDBFn
	origEnsure := ensureGooseVersionTableFn
	origAcquire := acquireMigrationLockFn

	return func() {
		openSQLDBFn = origOpen
		ensureGooseVersionTableFn = origEnsure
		acquireMigrationLockFn = origAcquire
	}
}
