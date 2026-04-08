package templates

import (
	"context"

	coreerrors "github.com/yogayulanda/go-core/errors"
)

// GRPCHandlerTemplate represents the transport layer in the golden path:
// validate request shape, map transport input/output, and convert to transport errors.
type ExecuteRequest struct {
	SubjectId string
	Amount    int64
}

type ExecuteResponse struct {
	Id string
}

type ServicePort interface {
	Execute(ctx context.Context, in ServiceInput) (ServiceOutput, error)
}

type GRPCHandlerTemplate struct {
	service ServicePort
}

func NewGRPCHandlerTemplate(service ServicePort) *GRPCHandlerTemplate {
	return &GRPCHandlerTemplate{service: service}
}

func (h *GRPCHandlerTemplate) Execute(ctx context.Context, req *ExecuteRequest) (*ExecuteResponse, error) {
	if req == nil {
		return nil, coreerrors.ToGRPC(coreerrors.Validation("invalid request", coreerrors.Detail{Field: "request", Reason: "required"}))
	}

	out, err := h.service.Execute(ctx, ServiceInput{SubjectID: req.SubjectId, Amount: req.Amount})
	if err != nil {
		return nil, coreerrors.ToGRPC(err)
	}

	return &ExecuteResponse{Id: out.ID}, nil
}
