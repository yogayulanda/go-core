package migration

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/pressly/goose/v3"
	"github.com/yogayulanda/go-core/config"
	"github.com/yogayulanda/go-core/logger"
)

type Runner interface {
	Run(db *sql.DB, driver string, dir string, action string) error
}

type GooseRunner struct{}

var defaultRunner Runner = GooseRunner{}
var openSQLDBFn = OpenSQLDB
var ensureGooseVersionTableFn = ensureGooseVersionTable
var acquireMigrationLockFn = acquireMigrationLock
var gooseUpFn = goose.Up
var gooseDownFn = goose.Down

func OpenSQLDB(cfg *config.Config, dbName string) (*sql.DB, config.DBConfig, error) {
	dbName = config.NormalizeDBAlias(dbName)
	dbCfg, ok := cfg.Databases[dbName]
	if !ok {
		for name, candidate := range cfg.Databases {
			if config.NormalizeDBAlias(name) == dbName {
				dbCfg = candidate
				ok = true
				break
			}
		}
		if !ok {
			return nil, config.DBConfig{}, fmt.Errorf("database %q not found in DB_LIST", dbName)
		}
	}

	db, err := sql.Open(dbCfg.Driver, dbCfg.DSN)
	if err != nil {
		return nil, config.DBConfig{}, fmt.Errorf("open db failed: %w", err)
	}

	return db, dbCfg, nil
}

func RunGoose(db *sql.DB, driver string, dir string, action string) error {
	return GooseRunner{}.Run(db, driver, dir, action)
}

func (GooseRunner) Run(db *sql.DB, driver string, dir string, action string) error {
	if err := goose.SetDialect(GooseDialect(driver)); err != nil {
		return fmt.Errorf("set goose dialect failed: %w", err)
	}

	switch strings.ToLower(strings.TrimSpace(action)) {
	case "up":
		if err := gooseUpFn(db, dir); err != nil && !errors.Is(err, goose.ErrNoNextVersion) {
			return fmt.Errorf("goose up failed: %w", err)
		}
	case "down":
		if err := gooseDownFn(db, dir); err != nil {
			return fmt.Errorf("goose down failed: %w", err)
		}
	default:
		return fmt.Errorf("invalid action %q, expected up or down", action)
	}

	return nil
}

func AutoRunUp(cfg *config.Config) error {
	return AutoRunUpWithRunnerAndLogger(cfg, defaultRunner, nil)
}

func AutoRunUpWithRunner(cfg *config.Config, runner Runner) error {
	return AutoRunUpWithRunnerAndLogger(cfg, runner, nil)
}

func AutoRunUpWithLogger(cfg *config.Config, log logger.Logger) error {
	return AutoRunUpWithRunnerAndLogger(cfg, defaultRunner, log)
}

func AutoRunUpWithRunnerAndLogger(cfg *config.Config, runner Runner, log logger.Logger) error {
	startedAt := time.Now()
	if !cfg.Migration.AutoRun {
		logMigration(context.Background(), log, "migration_autorun", "skipped", startedAt, "", map[string]interface{}{
			"auto_run": false,
		})
		return nil
	}
	if runner == nil {
		logMigration(context.Background(), log, "migration_autorun", "failed", startedAt, "runner_nil", map[string]interface{}{
			"auto_run": true,
		})
		return fmt.Errorf("migration runner is nil")
	}

	dbName := config.NormalizeDBAlias(cfg.Migration.DBName)
	db, dbCfg, err := openSQLDBFn(cfg, dbName)
	if err != nil {
		logMigration(context.Background(), log, "migration_autorun", "failed", startedAt, "open_db_failed", map[string]interface{}{
			"db_name": dbName,
		})
		return err
	}
	defer db.Close()

	if err := ensureGooseVersionTableFn(dbCfg.Driver, db); err != nil {
		logMigration(context.Background(), log, "migration_autorun", "failed", startedAt, "ensure_version_table_failed", map[string]interface{}{
			"db_name": dbName,
			"driver":  dbCfg.Driver,
		})
		return err
	}

	if cfg.Migration.LockEnabled {
		lockStartedAt := time.Now()
		lockKey := strings.TrimSpace(cfg.Migration.LockKey)
		if lockKey == "" {
			lockKey = defaultMigrationLockKey(cfg, dbName)
		}

		release, err := acquireMigrationLockFn(
			context.Background(),
			db,
			dbCfg.Driver,
			lockKey,
			cfg.Migration.LockTimeout,
		)
		if err != nil {
			logMigration(context.Background(), log, "migration_lock", "failed", lockStartedAt, "lock_acquire_failed", map[string]interface{}{
				"db_name":    dbName,
				"driver":     dbCfg.Driver,
				"lock_key":   lockKey,
				"timeout_ms": cfg.Migration.LockTimeout.Milliseconds(),
			})
			logMigration(context.Background(), log, "migration_autorun", "failed", startedAt, "lock_acquire_failed", map[string]interface{}{
				"db_name":  dbName,
				"driver":   dbCfg.Driver,
				"lock_key": lockKey,
			})
			return err
		}
		logMigration(context.Background(), log, "migration_lock", "success", lockStartedAt, "", map[string]interface{}{
			"db_name":    dbName,
			"driver":     dbCfg.Driver,
			"lock_key":   lockKey,
			"timeout_ms": cfg.Migration.LockTimeout.Milliseconds(),
		})
		defer func() {
			releaseStartedAt := time.Now()
			if err := release(context.Background()); err != nil {
				logMigration(context.Background(), log, "migration_lock", "failed", releaseStartedAt, "lock_release_failed", map[string]interface{}{
					"db_name":  dbName,
					"driver":   dbCfg.Driver,
					"lock_key": lockKey,
				})
				return
			}
			logMigration(context.Background(), log, "migration_lock", "released", releaseStartedAt, "", map[string]interface{}{
				"db_name":  dbName,
				"driver":   dbCfg.Driver,
				"lock_key": lockKey,
			})
		}()
	}

	if err := runner.Run(db, dbCfg.Driver, cfg.Migration.Dir, "up"); err != nil {
		logMigration(context.Background(), log, "migration_autorun", "failed", startedAt, "run_failed", map[string]interface{}{
			"db_name":      dbName,
			"driver":       dbCfg.Driver,
			"dir":          cfg.Migration.Dir,
			"lock_enabled": cfg.Migration.LockEnabled,
		})
		return err
	}

	logMigration(context.Background(), log, "migration_autorun", "success", startedAt, "", map[string]interface{}{
		"db_name":      dbName,
		"driver":       dbCfg.Driver,
		"dir":          cfg.Migration.Dir,
		"lock_enabled": cfg.Migration.LockEnabled,
	})
	return nil
}

func SetDefaultRunner(r Runner) {
	if r == nil {
		return
	}
	defaultRunner = r
}

func GooseDialect(driver string) string {
	switch strings.ToLower(strings.TrimSpace(driver)) {
	case "sqlserver":
		return "mssql"
	default:
		return strings.ToLower(strings.TrimSpace(driver))
	}
}

func ensureGooseVersionTable(driver string, db *sql.DB) error {
	if strings.ToLower(strings.TrimSpace(driver)) != "sqlserver" {
		return nil
	}

	const query = `
IF OBJECT_ID('dbo.goose_db_version', 'U') IS NULL
BEGIN
    CREATE TABLE dbo.goose_db_version (
        id BIGINT IDENTITY(1,1) NOT NULL PRIMARY KEY,
        version_id BIGINT NOT NULL,
        is_applied BIT NOT NULL,
        tstamp DATETIME2 NOT NULL DEFAULT SYSUTCDATETIME()
    );
END;

IF NOT EXISTS (SELECT 1 FROM dbo.goose_db_version)
BEGIN
    INSERT INTO dbo.goose_db_version (version_id, is_applied)
    VALUES (0, 1);
END;`

	if _, err := db.Exec(query); err != nil {
		return fmt.Errorf("ensure goose_db_version table failed: %w", err)
	}

	return nil
}

type releaseLockFunc func(ctx context.Context) error

func acquireMigrationLock(
	ctx context.Context,
	db *sql.DB,
	driver string,
	lockKey string,
	timeout time.Duration,
) (releaseLockFunc, error) {
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	driver = strings.ToLower(strings.TrimSpace(driver))

	switch driver {
	case "sqlserver":
		return acquireSQLServerLock(ctx, db, lockKey, timeout)
	case "mysql":
		return acquireMySQLLock(ctx, db, lockKey, timeout)
	case "postgres":
		return acquirePostgresLock(ctx, db, lockKey, timeout)
	default:
		// Graceful no-op for unsupported drivers.
		return func(context.Context) error { return nil }, nil
	}
}

func acquireSQLServerLock(
	ctx context.Context,
	db *sql.DB,
	lockKey string,
	timeout time.Duration,
) (releaseLockFunc, error) {
	lockCtx, cancel := context.WithTimeout(ctx, timeout+2*time.Second)
	defer cancel()

	var result int
	query := `
DECLARE @res INT;
EXEC @res = sp_getapplock
    @Resource = @p1,
    @LockMode = 'Exclusive',
    @LockOwner = 'Session',
    @LockTimeout = @p2;
SELECT @res;
`

	if err := db.QueryRowContext(
		lockCtx,
		query,
		lockKey,
		int(timeout.Milliseconds()),
	).Scan(&result); err != nil {
		return nil, fmt.Errorf("acquire sqlserver migration lock failed: %w", err)
	}
	if result < 0 {
		return nil, fmt.Errorf("acquire sqlserver migration lock failed with code %d", result)
	}

	return func(releaseCtx context.Context) error {
		_, err := db.ExecContext(
			releaseCtx,
			`EXEC sp_releaseapplock @Resource = @p1, @LockOwner = 'Session';`,
			lockKey,
		)
		return err
	}, nil
}

func acquireMySQLLock(
	ctx context.Context,
	db *sql.DB,
	lockKey string,
	timeout time.Duration,
) (releaseLockFunc, error) {
	lockCtx, cancel := context.WithTimeout(ctx, timeout+2*time.Second)
	defer cancel()

	var got sql.NullInt64
	waitSeconds := int(math.Ceil(timeout.Seconds()))
	if waitSeconds < 1 {
		waitSeconds = 1
	}

	if err := db.QueryRowContext(lockCtx, `SELECT GET_LOCK(?, ?)`, lockKey, waitSeconds).Scan(&got); err != nil {
		return nil, fmt.Errorf("acquire mysql migration lock failed: %w", err)
	}
	if !got.Valid || got.Int64 != 1 {
		return nil, fmt.Errorf("acquire mysql migration lock failed: lock timeout")
	}

	return func(releaseCtx context.Context) error {
		var _ignored sql.NullInt64
		return db.QueryRowContext(releaseCtx, `SELECT RELEASE_LOCK(?)`, lockKey).Scan(&_ignored)
	}, nil
}

func acquirePostgresLock(
	ctx context.Context,
	db *sql.DB,
	lockKey string,
	timeout time.Duration,
) (releaseLockFunc, error) {
	deadline := time.Now().Add(timeout)
	for {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}

		var locked bool
		err := db.QueryRowContext(ctx, `SELECT pg_try_advisory_lock(hashtext($1))`, lockKey).Scan(&locked)
		if err != nil {
			return nil, fmt.Errorf("acquire postgres migration lock failed: %w", err)
		}
		if locked {
			return func(releaseCtx context.Context) error {
				var _ignored bool
				return db.QueryRowContext(
					releaseCtx,
					`SELECT pg_advisory_unlock(hashtext($1))`,
					lockKey,
				).Scan(&_ignored)
			}, nil
		}

		if time.Now().After(deadline) {
			return nil, fmt.Errorf("acquire postgres migration lock failed: lock timeout")
		}
		time.Sleep(200 * time.Millisecond)
	}
}

func defaultMigrationLockKey(cfg *config.Config, dbName string) string {
	service := "service"
	if cfg != nil {
		service = strings.TrimSpace(cfg.App.ServiceName)
		if service == "" {
			service = "service"
		}
	}
	dbName = strings.TrimSpace(dbName)
	if dbName == "" {
		dbName = "default"
	}
	return fmt.Sprintf("%s:migration:%s", service, dbName)
}
