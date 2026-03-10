package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/yogayulanda/go-core/config"
	"github.com/yogayulanda/go-core/logger"
)

type DB struct {
	*sql.DB
}

// New initializes database connection pool.
func New(cfg config.DBConfig, log logger.Logger) (*DB, error) {

	db, err := sql.Open(cfg.Driver, cfg.DSN)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Pool configuration
	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)
	db.SetConnMaxLifetime(cfg.ConnMaxLifetime)

	// NEW: important for modern Go
	db.SetConnMaxIdleTime(2 * time.Minute)

	// Fail fast check
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	log.Info(ctx, "database connected",
		logger.Field{Key: "driver", Value: cfg.Driver},
		logger.Field{Key: "max_open_conns", Value: cfg.MaxOpenConns},
		logger.Field{Key: "max_idle_conns", Value: cfg.MaxIdleConns},
		logger.Field{Key: "conn_max_lifetime", Value: cfg.ConnMaxLifetime.String()},
	)

	return &DB{DB: db}, nil
}

// Health checks DB readiness.
func (d *DB) Health(ctx context.Context) error {
	return d.PingContext(ctx)
}

// Close gracefully closes database.
func (d *DB) Close() error {
	return d.DB.Close()
}
