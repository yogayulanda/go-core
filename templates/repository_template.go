package templates

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/yogayulanda/go-core/dbtx"
)

// SQLRepositoryTemplate represents the repository layer in the golden path:
// reuse `dbtx.FromContext(ctx)` when the service layer has already opened a transaction.
type sqlExecutor interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

type SQLRepositoryTemplate struct {
	db     *sql.DB
	driver string
}

func NewSQLRepositoryTemplate(db *sql.DB, driver string) *SQLRepositoryTemplate {
	return &SQLRepositoryTemplate{db: db, driver: strings.ToLower(strings.TrimSpace(driver))}
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
	query := `INSERT INTO entities (id, subject_id, amount, created_at) VALUES (?, ?, ?, ?)`
	if r.driver == "sqlserver" {
		query = `INSERT INTO entities (id, subject_id, amount, created_at) VALUES (@p1, @p2, @p3, @p4)`
	} else if r.driver != "mysql" {
		return ServiceOutput{}, fmt.Errorf("unsupported SQL template driver %q", r.driver)
	}
	_, err := exec.ExecContext(
		ctx,
		query,
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
