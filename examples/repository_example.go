package examples

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/yogayulanda/go-core/dbtx"
	"github.com/yogayulanda/go-core/logger"
)

type sqlExec interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

type RecordSQLRepository struct {
	db     *sql.DB
	dbName string
	log    logger.Logger
}

func NewRecordSQLRepository(db *sql.DB, dbName string, log logger.Logger) *RecordSQLRepository {
	return &RecordSQLRepository{db: db, dbName: dbName, log: log}
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
	startedAt := time.Now()
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
		if r.log != nil {
			r.log.LogDB(ctx, logger.DBLog{
				Operation:  "record_insert",
				DBName:     r.dbName,
				Status:     "failed",
				DurationMs: time.Since(startedAt).Milliseconds(),
				ErrorCode:  "insert_failed",
				Metadata: map[string]interface{}{
					"reference_id": in.ReferenceID,
					"error":        err.Error(),
				},
			})
		}
		return "", fmt.Errorf("insert record failed: %w", err)
	}

	if r.log != nil {
		r.log.LogDB(ctx, logger.DBLog{
			Operation:  "record_insert",
			DBName:     r.dbName,
			Status:     "success",
			DurationMs: time.Since(startedAt).Milliseconds(),
			Metadata: map[string]interface{}{
				"record_id": id,
			},
		})
	}
	return id, nil
}
