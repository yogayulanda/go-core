package migration

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/pressly/goose/v3"
	"github.com/yogayulanda/go-core/config"
)

type Runner interface {
	Run(db *sql.DB, driver string, dir string, action string) error
}

type GooseRunner struct{}

var defaultRunner Runner = GooseRunner{}
var openSQLDBFn = OpenSQLDB
var ensureGooseVersionTableFn = ensureGooseVersionTable
var acquireMigrationLockFn = acquireMigrationLock

func OpenSQLDB(cfg *config.Config, dbName string) (*sql.DB, config.DBConfig, error) {
	dbCfg, ok := cfg.Databases[dbName]
	if !ok {
		return nil, config.DBConfig{}, fmt.Errorf("database %q not found in DB_LIST", dbName)
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
		if err := goose.Up(db, dir); err != nil {
			return fmt.Errorf("goose up failed: %w", err)
		}
	case "down":
		if err := goose.Down(db, dir); err != nil {
			return fmt.Errorf("goose down failed: %w", err)
		}
	default:
		return fmt.Errorf("invalid action %q, expected up or down", action)
	}

	return nil
}

func AutoRunUp(cfg *config.Config) error {
	return AutoRunUpWithRunner(cfg, defaultRunner)
}

func AutoRunUpWithRunner(cfg *config.Config, runner Runner) error {
	if !cfg.Migration.AutoRun {
		return nil
	}
	if runner == nil {
		return fmt.Errorf("migration runner is nil")
	}

	dbName := strings.ToLower(strings.TrimSpace(cfg.Migration.DBName))
	db, dbCfg, err := openSQLDBFn(cfg, dbName)
	if err != nil {
		return err
	}
	defer db.Close()

	if err := ensureGooseVersionTableFn(dbCfg.Driver, db); err != nil {
		return err
	}

	if cfg.Migration.LockEnabled {
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
			return err
		}
		defer func() {
			_ = release(context.Background())
		}()
	}

	return runner.Run(db, dbCfg.Driver, cfg.Migration.Dir, "up")
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
