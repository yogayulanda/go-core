package database

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/yogayulanda/go-core/config"
	"github.com/yogayulanda/go-core/logger"
	"gorm.io/driver/mysql"
	"gorm.io/driver/sqlserver"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

type DB struct {
	*sql.DB
	gorm *gorm.DB
}

// New initializes database connection pool and GORM instance.
func New(cfg config.DBConfig, log logger.Logger) (*DB, error) {
	return NewWithLogConfig(cfg, log, time.Second)
}

func NewWithLogConfig(cfg config.DBConfig, log logger.Logger, slowQueryThreshold time.Duration) (*DB, error) {
	startedAt := time.Now()
	if _, err := gormDialector(cfg.Driver, nil); err != nil {
		return nil, err
	}

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

	// Initialize GORM
	dialector, err := gormDialector(cfg.Driver, db)
	if err != nil {
		return nil, err
	}
	gormDB, err := gorm.Open(dialector, &gorm.Config{
		PrepareStmt: true,
		Logger:      logger.NewGORMLogger(log, slowQueryThreshold),
		NamingStrategy: schema.NamingStrategy{
			SingularTable: true,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize gorm: %w", err)
	}

	log.LogDB(ctx, logger.DBLog{
		Operation:  "connect",
		DBName:     cfg.Name,
		Status:     "success",
		DurationMs: time.Since(startedAt).Milliseconds(),
		Metadata: map[string]interface{}{
			"driver":           cfg.Driver,
			"max_open_conns":   cfg.MaxOpenConns,
			"max_idle_conns":   cfg.MaxIdleConns,
			"gorm_initialized": true,
		},
	})

	return &DB{DB: db, gorm: gormDB}, nil
}

func gormDialector(driver string, conn gorm.ConnPool) (gorm.Dialector, error) {
	switch strings.ToLower(strings.TrimSpace(driver)) {
	case "sqlserver":
		return sqlserver.New(sqlserver.Config{Conn: conn}), nil
	case "mysql":
		return mysql.New(mysql.Config{Conn: conn}), nil
	default:
		return nil, fmt.Errorf("unsupported database driver %q (supported: sqlserver, mysql)", driver)
	}
}

// Gorm returns the pre-initialized GORM DB instance.
func (d *DB) Gorm() *gorm.DB {
	return d.gorm
}

// Health checks DB readiness.
func (d *DB) Health(ctx context.Context) error {
	return d.PingContext(ctx)
}

// Close gracefully closes database.
func (d *DB) Close() error {
	return d.DB.Close()
}
