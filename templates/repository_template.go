package templates

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/yogayulanda/go-core/dbtx"
)

// SQLRepositoryTemplate represents the repository layer in the golden path:
// reuse `dbtx.FromContext(ctx)` when the service layer has already opened a transaction.
type sqlExecutor interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

type SQLRepositoryTemplate struct {
	db *sql.DB
}

func NewSQLRepositoryTemplate(db *sql.DB) *SQLRepositoryTemplate {
	return &SQLRepositoryTemplate{db: db}
}

func (r *SQLRepositoryTemplate) Create(ctx context.Context, in ServiceInput) (ServiceOutput, error) {
	if r == nil || r.db == nil {
		return ServiceOutput{}, errors.New("sql repository not initialized")
	}

	exec := sqlExecutor(r.db)
	if tx, ok := dbtx.FromContext(ctx); ok && tx != nil {
		exec = tx
	}

	id := fmt.Sprintf("id-%d", time.Now().UnixNano())
	_, err := exec.ExecContext(
		ctx,
		`INSERT INTO entities (id, subject_id, amount, created_at) VALUES (?, ?, ?, ?)`,
		id,
		in.SubjectID,
		in.Amount,
		time.Now().UTC(),
	)
	if err != nil {
		return ServiceOutput{}, fmt.Errorf("insert failed: %w", err)
	}

	return ServiceOutput{ID: id}, nil
}
