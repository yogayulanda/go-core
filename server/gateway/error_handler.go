package gateway

import (
	"context"
	"encoding/json"
	stderrors "errors"
	"net/http"
	"time"

	"github.com/yogayulanda/go-core/app"
	"github.com/yogayulanda/go-core/observability"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"go.opentelemetry.io/otel/trace"

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

		traceID := requestID
		if span := trace.SpanFromContext(ctx); span != nil && span.SpanContext().IsValid() {
			traceID = span.SpanContext().TraceID().String()
		}

		txID := observability.GetGTID(ctx)
		resp, httpStatus := legacyErrorResponseFromError(err, traceID, txID)
		w.WriteHeader(httpStatus)
		resp.Timestamp = time.Now().UTC().Format(time.RFC3339)

		_ = json.NewEncoder(w).Encode(resp)
	}
}

func legacyErrorResponseFromError(err error, traceID, transactionID string) (legacyGatewayError, int) {
	response := legacyGatewayError{
		Success:       false,
		TraceID:       traceID,
		TransactionID: transactionID,
	}
	response.Details = nil
	response.Code = "CORE-9000"
	response.Message = "internal server error"
	statusCode := http.StatusInternalServerError

	var appErr *coreErrors.AppError
	if stderrors.As(err, &appErr) {
		def, registered := coreErrors.Resolve(appErr.Code)
		response.Code = def.Code
		response.Message = def.DefaultMessage
		statusCode = def.HTTPStatus
		if registered && def.Code == "CORE-1000" {
			response.Details = appErr.Details
		}
		return response, statusCode
	}

	if info, ok := coreErrors.InfoFromGRPC(err); ok {
		def, _ := coreErrors.Lookup(info.Code)
		response.Code = def.Code
		response.Message = def.DefaultMessage
		statusCode = def.HTTPStatus
		if def.Code == "CORE-1000" {
			response.Details = coreErrors.DetailsFromGRPC(err)
		}
	}
	return response, statusCode
}
