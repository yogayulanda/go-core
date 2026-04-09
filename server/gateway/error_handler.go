package gateway

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/yogayulanda/go-core/app"
	"github.com/yogayulanda/go-core/observability"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc/codes"
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

		httpStatus := httpStatusFromError(err)
		w.WriteHeader(httpStatus)

		_ = json.NewEncoder(w).Encode(coreErrors.ErrorResponseFromError(err, requestID))
	}
}

func httpStatusFromError(err error) int {
	st, ok := status.FromError(err)
	if ok {
		return runtime.HTTPStatusFromCode(st.Code())
	}

	if resp := coreErrors.ErrorResponseFromError(err, ""); resp.Code != "" {
		return runtime.HTTPStatusFromCode(grpcStatusCodeFromCoreCode(coreErrors.Code(resp.Code)))
	}

	return http.StatusInternalServerError
}

func grpcStatusCodeFromCoreCode(code coreErrors.Code) codes.Code {
	switch code {
	case coreErrors.CodeInvalidRequest:
		return codes.InvalidArgument
	case coreErrors.CodeUnauthorized, coreErrors.CodeSessionExpired:
		return codes.Unauthenticated
	case coreErrors.CodeForbidden:
		return codes.PermissionDenied
	case coreErrors.CodeNotFound:
		return codes.NotFound
	case coreErrors.CodeServiceUnavailable:
		return codes.Unavailable
	default:
		return codes.Internal
	}
}
