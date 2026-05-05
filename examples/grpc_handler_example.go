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
	// Transport-level request ID, metrics, and structured request logging
	// are provided by go-core gRPC interceptors, not by handler code.
	if req == nil {
		err := coreerrors.Build("EXM", coreerrors.CategoryVAL, "001").
			Message("invalid request").
			UserMessage("Permintaan tidak valid").
			Finality(coreerrors.FinalityBusiness).
			Details(coreerrors.Detail{Field: "request", Reason: "required"}).
			Done()
		return nil, coreerrors.ToGRPC(err)
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
