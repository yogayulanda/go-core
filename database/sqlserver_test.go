package database

import (
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/yogayulanda/go-core/config"
	"github.com/yogayulanda/go-core/logger"
)

func TestNew_InvalidDriver_ReturnError(t *testing.T) {
	log, err := logger.New("db-test", "error")
	if err != nil {
		t.Fatalf("init logger failed: %v", err)
	}

	_, err = New(config.DBConfig{
		Driver:          "invalid-driver",
		DSN:             "dummy",
		MaxOpenConns:    1,
		MaxIdleConns:    1,
		ConnMaxLifetime: time.Minute,
	}, log)
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestNew_SQLMock_Success(t *testing.T) {
	const dsn = "go_core_database_sqlmock_success"

	rawDB, mock, err := sqlmock.NewWithDSN(dsn, sqlmock.MonitorPingsOption(true))
	if err != nil {
		t.Fatalf("create sqlmock failed: %v", err)
	}
	defer rawDB.Close()

	mock.ExpectPing()
	mock.ExpectPing()

	log, err := logger.New("db-test", "error")
	if err != nil {
		t.Fatalf("init logger failed: %v", err)
	}

	db, err := New(config.DBConfig{
		Driver:          "sqlmock",
		DSN:             dsn,
		MaxOpenConns:    2,
		MaxIdleConns:    1,
		ConnMaxIdleTime: 2 * time.Minute,
		ConnMaxLifetime: time.Minute,
	}, log)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if db == nil {
		t.Fatalf("expected db")
	}
	_ = db.Close()

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestNew_UsesConfiguredConnMaxIdleTime(t *testing.T) {
	const dsn = "go_core_database_sqlmock_idle_time"

	rawDB, mock, err := sqlmock.NewWithDSN(dsn, sqlmock.MonitorPingsOption(true))
	if err != nil {
		t.Fatalf("create sqlmock failed: %v", err)
	}
	defer rawDB.Close()

	mock.ExpectPing()
	mock.ExpectPing()

	log, err := logger.New("db-test", "error")
	if err != nil {
		t.Fatalf("init logger failed: %v", err)
	}

	db, err := New(config.DBConfig{
		Driver:          "sqlmock",
		DSN:             dsn,
		MaxOpenConns:    2,
		MaxIdleConns:    1,
		ConnMaxIdleTime: 3 * time.Minute,
		ConnMaxLifetime: time.Minute,
	}, log)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	stats := db.DB.Stats()
	if stats.MaxIdleClosed != 0 {
		t.Fatalf("unexpected stats mutation: %+v", stats)
	}

	_ = db.Close()
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}
