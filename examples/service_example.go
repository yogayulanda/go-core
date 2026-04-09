package examples

import (
	"context"
	"strings"
	"time"

	coreerrors "github.com/yogayulanda/go-core/errors"
	"github.com/yogayulanda/go-core/logger"
)

type CreateRecordInput struct {
	SubjectID   string
	ReferenceID string
	Amount      int64
}

type RecordRepository interface {
	Create(ctx context.Context, in CreateRecordInput) (string, error)
}

type RecordService struct {
	repo RecordRepository
	log  logger.Logger
}

func NewRecordService(repo RecordRepository, log logger.Logger) *RecordService {
	return &RecordService{repo: repo, log: log}
}

func (s *RecordService) Create(ctx context.Context, in CreateRecordInput) (string, error) {
	startedAt := time.Now()
	if strings.TrimSpace(in.SubjectID) == "" {
		return "", coreerrors.Validation("invalid request", coreerrors.Detail{Field: "subject_id", Reason: "required"})
	}
	if strings.TrimSpace(in.ReferenceID) == "" {
		return "", coreerrors.Validation("invalid request", coreerrors.Detail{Field: "reference_id", Reason: "required"})
	}
	if in.Amount <= 0 {
		return "", coreerrors.Validation("invalid request", coreerrors.Detail{Field: "amount", Reason: "must be > 0"})
	}

	id, err := s.repo.Create(ctx, in)
	if err != nil {
		if s.log != nil {
			s.log.LogService(ctx, logger.ServiceLog{
				Operation:  "record_create",
				Status:     "failed",
				DurationMs: time.Since(startedAt).Milliseconds(),
				ErrorCode:  "repository_error",
				Metadata: map[string]interface{}{
					"reference_id": in.ReferenceID,
					"error":        err.Error(),
				},
			})
		}
		return "", coreerrors.Wrap(coreerrors.CodeInternal, "create record failed", err)
	}

	if s.log != nil {
		s.log.LogService(ctx, logger.ServiceLog{
			Operation:  "record_create",
			Status:     "success",
			DurationMs: time.Since(startedAt).Milliseconds(),
			Metadata: map[string]interface{}{
				"record_id":    id,
				"reference_id": in.ReferenceID,
				"amount":       in.Amount,
			},
		})
	}
	return id, nil
}
