package config

import (
	"testing"
	"time"
)

func TestLoad_DatabaseAliasWithUnderscoreAndIdleTime(t *testing.T) {
	t.Setenv("SERVICE_NAME", "svc")
	t.Setenv("DB_LIST", "transaction_history")
	t.Setenv("DB_TRANSACTION_HISTORY_DRIVER", "sqlserver")
	t.Setenv("DB_TRANSACTION_HISTORY_DSN", "sqlserver://user:pass@localhost:1433?database=test")
	t.Setenv("DB_TRANSACTION_HISTORY_CONN_MAX_IDLE_TIME", "7m")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	db, ok := cfg.Databases["transaction_history"]
	if !ok {
		t.Fatalf("expected normalized alias transaction_history")
	}
	if db.Driver != "sqlserver" {
		t.Fatalf("unexpected driver: %q", db.Driver)
	}
	if db.ConnMaxIdleTime != 7*time.Minute {
		t.Fatalf("unexpected conn max idle time: %v", db.ConnMaxIdleTime)
	}
}

func TestLoad_MigrationDefaultsAreExplicit(t *testing.T) {
	t.Setenv("SERVICE_NAME", "svc")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Migration.DBName != "" {
		t.Fatalf("expected empty migration db default, got %q", cfg.Migration.DBName)
	}
	if cfg.Migration.Dir != "" {
		t.Fatalf("expected empty migration dir default, got %q", cfg.Migration.Dir)
	}
}
