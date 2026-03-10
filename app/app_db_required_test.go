package app

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/yogayulanda/go-core/config"
	"github.com/yogayulanda/go-core/database"
	"github.com/yogayulanda/go-core/logger"
)

func TestNew_RequiredDatabaseFailure_ReturnError(t *testing.T) {
	restore := overrideNewDatabaseFn(t)
	defer restore()

	newDatabaseFn = func(cfg config.DBConfig, log logger.Logger) (*database.DB, error) {
		return nil, errors.New("db connect failed")
	}

	cfg := baseAppConfig()
	cfg.Databases = map[string]config.DBConfig{
		"transaction": {
			Driver:   "mysql",
			DSN:      "dummy",
			Required: true,
		},
	}

	_, err := New(context.Background(), cfg)
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestNew_OptionalDatabaseFailure_ContinueStartup(t *testing.T) {
	restore := overrideNewDatabaseFn(t)
	defer restore()

	rawDB, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("create sqlmock failed: %v", err)
	}
	defer rawDB.Close()

	newDatabaseFn = func(cfg config.DBConfig, log logger.Logger) (*database.DB, error) {
		if cfg.Driver == "sqlserver" {
			return nil, errors.New("optional db down")
		}
		return &database.DB{DB: rawDB}, nil
	}

	cfg := baseAppConfig()
	cfg.Databases = map[string]config.DBConfig{
		"transaction": {
			Driver:   "mysql",
			DSN:      "dummy",
			Required: true,
		},
		"history": {
			Driver:   "sqlserver",
			DSN:      "dummy",
			Required: false,
		},
	}

	application, err := New(context.Background(), cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	all := application.SQLAll()
	if len(all) != 1 {
		t.Fatalf("expected 1 available db, got %d", len(all))
	}
	if application.SQLByName("transaction") == nil {
		t.Fatalf("required db must exist")
	}
	if application.SQLByName("history") != nil {
		t.Fatalf("optional failed db must be omitted")
	}
}

func TestNew_AllOptionalDatabaseFailure_StartupSuccess(t *testing.T) {
	restore := overrideNewDatabaseFn(t)
	defer restore()

	newDatabaseFn = func(cfg config.DBConfig, log logger.Logger) (*database.DB, error) {
		return nil, errors.New("db unavailable")
	}

	cfg := baseAppConfig()
	cfg.Databases = map[string]config.DBConfig{
		"history": {
			Driver:   "sqlserver",
			DSN:      "dummy",
			Required: false,
		},
	}

	application, err := New(context.Background(), cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(application.SQLAll()) != 0 {
		t.Fatalf("expected no db connections")
	}
}

func baseAppConfig() *config.Config {
	return &config.Config{
		App: config.AppConfig{
			ServiceName:     "app-db-test",
			Environment:     "test",
			LogLevel:        "error",
			ShutdownTimeout: time.Second,
		},
		Databases: map[string]config.DBConfig{},
	}
}

func overrideNewDatabaseFn(t *testing.T) func() {
	t.Helper()
	orig := newDatabaseFn
	return func() {
		newDatabaseFn = orig
	}
}
