package examples

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/yogayulanda/go-core/dbtx"
)

type sqlExec interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

type RecordSQLRepository struct {
	db *sql.DB
}

func NewRecordSQLRepository(db *sql.DB) *RecordSQLRepository {
	return &RecordSQLRepository{db: db}
}

func (r *RecordSQLRepository) Create(ctx context.Context, in CreateRecordInput) (string, error) {
	if r == nil || r.db == nil {
		return "", errors.New("repository db is nil")
	}

	exec := sqlExec(r.db)
	if tx, ok := dbtx.FromContext(ctx); ok && tx != nil {
		exec = tx
	}

	id := fmt.Sprintf("rec-%d", time.Now().UnixNano())
	_, err := exec.ExecContext(
		ctx,
		`INSERT INTO records (id, subject_id, reference_id, amount, created_at) VALUES (?, ?, ?, ?, ?)`,
		id,
		in.SubjectID,
		in.ReferenceID,
		in.Amount,
		time.Now().UTC(),
	)
	if err != nil {
		return "", fmt.Errorf("insert record failed: %w", err)
	}

	return id, nil
}
