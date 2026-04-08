package templates

import (
	"context"
	"strings"

	coreerrors "github.com/yogayulanda/go-core/errors"
)

// ServiceTemplate represents the service/use-case layer in the golden path:
// validate input, define transaction boundary, call repository ports, and return app errors.
type ServiceInput struct {
	SubjectID string
	Amount    int64
}

type ServiceOutput struct {
	ID string
}

type RepositoryPort interface {
	Create(ctx context.Context, in ServiceInput) (ServiceOutput, error)
}

type ServiceTemplate struct {
	repo RepositoryPort
}

func NewServiceTemplate(repo RepositoryPort) *ServiceTemplate {
	return &ServiceTemplate{repo: repo}
}

func (s *ServiceTemplate) Execute(ctx context.Context, in ServiceInput) (ServiceOutput, error) {
	if strings.TrimSpace(in.SubjectID) == "" {
		return ServiceOutput{}, coreerrors.Validation("invalid request", coreerrors.Detail{Field: "subject_id", Reason: "required"})
	}
	if in.Amount <= 0 {
		return ServiceOutput{}, coreerrors.Validation("invalid request", coreerrors.Detail{Field: "amount", Reason: "must be > 0"})
	}

	out, err := s.repo.Create(ctx, in)
	if err != nil {
		return ServiceOutput{}, err
	}

	return out, nil
}
