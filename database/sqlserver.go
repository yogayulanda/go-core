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
	startedAt := time.Now()

	db, err := sql.Open(cfg.Driver, cfg.DSN)
	if err != nil {
		log.LogDB(context.Background(), logger.DBLog{
			Operation:  "connect",
			DBName:     cfg.Name,
			Status:     "failed",
			DurationMs: time.Since(startedAt).Milliseconds(),
			ErrorCode:  "open_failed",
			Metadata: map[string]interface{}{
				"driver": cfg.Driver,
			},
		})
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Pool configuration
	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)
	db.SetConnMaxLifetime(cfg.ConnMaxLifetime)
	db.SetConnMaxIdleTime(cfg.ConnMaxIdleTime)

	// Fail fast check
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		log.LogDB(ctx, logger.DBLog{
			Operation:  "connect",
			DBName:     cfg.Name,
			Status:     "failed",
			DurationMs: time.Since(startedAt).Milliseconds(),
			ErrorCode:  "ping_failed",
			Metadata: map[string]interface{}{
				"driver": cfg.Driver,
			},
		})
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	log.LogDB(ctx, logger.DBLog{
		Operation:  "connect",
		DBName:     cfg.Name,
		Status:     "success",
		DurationMs: time.Since(startedAt).Milliseconds(),
		Metadata: map[string]interface{}{
			"driver":             cfg.Driver,
			"max_open_conns":     cfg.MaxOpenConns,
			"max_idle_conns":     cfg.MaxIdleConns,
			"conn_max_idle_time": cfg.ConnMaxIdleTime.String(),
			"conn_max_lifetime":  cfg.ConnMaxLifetime.String(),
		},
	})

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
