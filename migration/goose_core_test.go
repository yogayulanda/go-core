package migration

import (
	"database/sql"
	"errors"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/yogayulanda/go-core/config"
)

func TestOpenSQLDB_MissingDatabase_ReturnNotFoundError(t *testing.T) {
	cfg := &config.Config{
		Databases: map[string]config.DBConfig{},
	}

	_, _, err := OpenSQLDB(cfg, "missing")
	if err == nil {
		t.Fatalf("expected error")
	}
	if !strings.Contains(err.Error(), "not found in DB_LIST") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestOpenSQLDB_InvalidDriver_ReturnOpenError(t *testing.T) {
	cfg := &config.Config{
		Databases: map[string]config.DBConfig{
			"transaction": {
				Driver: "unknown-driver",
				DSN:    "dummy",
			},
		},
	}

	_, _, err := OpenSQLDB(cfg, "transaction")
	if err == nil {
		t.Fatalf("expected error")
	}
	if !strings.Contains(err.Error(), "open db failed") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestOpenSQLDB_ValidSQLMockConfig_ReturnDB(t *testing.T) {
	const dsn = "go_core_migration_sqlmock_open"

	mockDB, mock, err := sqlmock.NewWithDSN(dsn)
	if err != nil {
		t.Fatalf("create sqlmock with dsn failed: %v", err)
	}
	defer mockDB.Close()

	cfg := &config.Config{
		Databases: map[string]config.DBConfig{
			"transaction": {
				Driver: "sqlmock",
				DSN:    dsn,
			},
		},
	}

	db, dbCfg, err := OpenSQLDB(cfg, "transaction")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if db == nil {
		t.Fatalf("expected db")
	}
	if dbCfg.Driver != "sqlmock" {
		t.Fatalf("unexpected driver: %s", dbCfg.Driver)
	}
	_ = db.Close()

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}

func TestGooseDialect_Driver_ReturnExpectedDialect(t *testing.T) {
	if got := GooseDialect("sqlserver"); got != "mssql" {
		t.Fatalf("expected mssql, got %q", got)
	}
	if got := GooseDialect(" MySQL "); got != "mysql" {
		t.Fatalf("expected mysql, got %q", got)
	}
}

func TestRunGoose_InvalidAction_ReturnError(t *testing.T) {
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("create sqlmock failed: %v", err)
	}
	defer db.Close()

	err = RunGoose(db, "mysql", ".", "noop")
	if err == nil {
		t.Fatalf("expected error")
	}
	if !strings.Contains(err.Error(), "invalid action") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGooseRunnerRun_UnknownDriver_ReturnDialectError(t *testing.T) {
	err := GooseRunner{}.Run(nil, "unknown-driver", ".", "up")
	if err == nil {
		t.Fatalf("expected error")
	}
	if !strings.Contains(err.Error(), "set goose dialect failed") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGooseRunnerRun_InvalidMigrationDirUp_ReturnError(t *testing.T) {
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("create sqlmock failed: %v", err)
	}
	defer db.Close()

	err = GooseRunner{}.Run(db, "mysql", "/path/that/does/not/exist", "up")
	if err == nil {
		t.Fatalf("expected error")
	}
	if !strings.Contains(err.Error(), "goose up failed") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGooseRunnerRun_InvalidMigrationDirDown_ReturnError(t *testing.T) {
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("create sqlmock failed: %v", err)
	}
	defer db.Close()

	err = GooseRunner{}.Run(db, "mysql", "/path/that/does/not/exist", "down")
	if err == nil {
		t.Fatalf("expected error")
	}
	if !strings.Contains(err.Error(), "goose down failed") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSetDefaultRunner_Runner_UpdateOrKeepCurrent(t *testing.T) {
	orig := defaultRunner
	defer func() { defaultRunner = orig }()

	runner := &fakeRunner{}
	SetDefaultRunner(runner)
	if defaultRunner != runner {
		t.Fatalf("expected default runner changed")
	}

	SetDefaultRunner(nil)
	if defaultRunner != runner {
		t.Fatalf("default runner should not change when nil provided")
	}
}

func TestAutoRunUp_AutoRunEnabled_UseDefaultRunner(t *testing.T) {
	restore := overrideMigrationDeps(t)
	defer restore()

	orig := defaultRunner
	defer func() { defaultRunner = orig }()

	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("create sqlmock failed: %v", err)
	}
	defer db.Close()

	openSQLDBFn = func(_ *config.Config, _ string) (*sql.DB, config.DBConfig, error) {
		return db, config.DBConfig{Driver: "mysql"}, nil
	}
	ensureGooseVersionTableFn = func(_ string, _ *sql.DB) error { return nil }

	runner := &fakeRunner{}
	defaultRunner = runner

	cfg := minimalAutoRunConfig()
	cfg.Migration.LockEnabled = false

	if err := AutoRunUp(cfg); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !runner.called {
		t.Fatalf("expected default runner to be called")
	}
}

func TestEnsureGooseVersionTable_NonSQLServer_Noop(t *testing.T) {
	if err := ensureGooseVersionTable("mysql", nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestEnsureGooseVersionTable_SQLServer_CreateIfMissing(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("create sqlmock failed: %v", err)
	}
	defer db.Close()

	mock.ExpectExec(`OBJECT_ID\('dbo\.goose_db_version', 'U'\)`).
		WillReturnResult(sqlmock.NewResult(0, 0))

	if err := ensureGooseVersionTable("sqlserver", db); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}

func TestEnsureGooseVersionTable_SQLServerExecError_ReturnError(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("create sqlmock failed: %v", err)
	}
	defer db.Close()

	mock.ExpectExec(`OBJECT_ID\('dbo\.goose_db_version', 'U'\)`).
		WillReturnError(errors.New("exec failed"))

	err = ensureGooseVersionTable("sqlserver", db)
	if err == nil {
		t.Fatalf("expected error")
	}
	if !strings.Contains(err.Error(), "ensure goose_db_version table failed") {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet sql expectations: %v", err)
	}
}
