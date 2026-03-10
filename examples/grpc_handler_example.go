package examples

import (
	"context"

	coreerrors "github.com/yogayulanda/go-core/errors"
)

type CreateRecordRequest struct {
	SubjectId   string
	ReferenceId string
	Amount      int64
}

type CreateRecordResponse struct {
	Id string
}

type RecordUseCase interface {
	Create(ctx context.Context, in CreateRecordInput) (string, error)
}

type RecordGRPCHandler struct {
	uc RecordUseCase
}

func NewRecordGRPCHandler(uc RecordUseCase) *RecordGRPCHandler {
	return &RecordGRPCHandler{uc: uc}
}

func (h *RecordGRPCHandler) CreateRecord(ctx context.Context, req *CreateRecordRequest) (*CreateRecordResponse, error) {
	if req == nil {
		return nil, coreerrors.ToGRPC(coreerrors.Validation("invalid request", coreerrors.Detail{Field: "request", Reason: "required"}))
	}

	id, err := h.uc.Create(ctx, CreateRecordInput{
		SubjectID:   req.SubjectId,
		ReferenceID: req.ReferenceId,
		Amount:      req.Amount,
	})
	if err != nil {
		return nil, coreerrors.ToGRPC(err)
	}

	return &CreateRecordResponse{Id: id}, nil
}
