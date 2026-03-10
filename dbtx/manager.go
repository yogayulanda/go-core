package dbtx

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

type Beginner interface {
	BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error)
}

// WithTx wraps function execution in a database transaction using default options.
func WithTx(
	ctx context.Context,
	db Beginner,
	fn func(txCtx context.Context) error,
) error {
	return WithTxOptions(ctx, db, nil, fn)
}

// WithTxOptions wraps function execution in a database transaction using custom options.
func WithTxOptions(
	ctx context.Context,
	db Beginner,
	opts *sql.TxOptions,
	fn func(txCtx context.Context) error,
) (err error) {
	if db == nil {
		return errors.New("dbtx: db is nil")
	}
	if fn == nil {
		return errors.New("dbtx: fn is nil")
	}

	tx, err := db.BeginTx(ctx, opts)
	if err != nil {
		return fmt.Errorf("dbtx: begin tx failed: %w", err)
	}

	committed := false
	defer func() {
		if r := recover(); r != nil {
			_ = tx.Rollback()
			panic(r)
		}

		if committed {
			return
		}

		if rollbackErr := tx.Rollback(); rollbackErr != nil && !errors.Is(rollbackErr, sql.ErrTxDone) {
			if err == nil {
				err = fmt.Errorf("dbtx: rollback failed: %w", rollbackErr)
			} else {
				err = errors.Join(err, fmt.Errorf("dbtx: rollback failed: %w", rollbackErr))
			}
		}
	}()

	txCtx := Inject(ctx, tx)
	if err = fn(txCtx); err != nil {
		return err
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("dbtx: commit failed: %w", err)
	}
	committed = true
	return nil
}
