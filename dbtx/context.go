package dbtx

import (
	"context"
	"database/sql"
)

type contextKey string

const txContextKey contextKey = "dbtx_sql_tx"

// Inject stores sql transaction in context.
func Inject(ctx context.Context, tx *sql.Tx) context.Context {
	return context.WithValue(ctx, txContextKey, tx)
}

// FromContext extracts sql transaction from context.
func FromContext(ctx context.Context) (*sql.Tx, bool) {
	tx, ok := ctx.Value(txContextKey).(*sql.Tx)
	return tx, ok
}
