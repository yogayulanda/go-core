package gateway

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/yogayulanda/go-core/app"
	"github.com/yogayulanda/go-core/observability"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc/status"

	coreErrors "github.com/yogayulanda/go-core/errors"
)

func customErrorHandler(app *app.App) runtime.ErrorHandlerFunc {
	return func(
		ctx context.Context,
		mux *runtime.ServeMux,
		marshaler runtime.Marshaler,
		w http.ResponseWriter,
		r *http.Request,
		err error,
	) {
		requestID := observability.GetRequestID(ctx)
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("x-request-id", requestID)

		st, ok := status.FromError(err)
		if !ok {
			w.WriteHeader(http.StatusInternalServerError)
			_ = json.NewEncoder(w).Encode(coreErrors.ErrorResponse{
				Code:      string(coreErrors.CodeInternal),
				Message:   "internal server error",
				RequestID: requestID,
			})
			return
		}

		httpStatus := runtime.HTTPStatusFromCode(st.Code())
		w.WriteHeader(httpStatus)

		resp := coreErrors.ErrorResponse{
			Code:      string(coreErrors.CodeFromGRPC(err)),
			Message:   st.Message(),
			RequestID: requestID,
			Details:   coreErrors.DetailsFromGRPC(err),
		}

		_ = json.NewEncoder(w).Encode(resp)
	}
}
