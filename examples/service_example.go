package examples

import (
	"context"
	"strings"

	coreerrors "github.com/yogayulanda/go-core/errors"
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
}

func NewRecordService(repo RecordRepository) *RecordService {
	return &RecordService{repo: repo}
}

func (s *RecordService) Create(ctx context.Context, in CreateRecordInput) (string, error) {
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
		return "", coreerrors.Wrap(coreerrors.CodeInternal, "create record failed", err)
	}

	return id, nil
}
