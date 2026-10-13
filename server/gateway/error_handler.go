package gateway

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/yogayulanda/go-core/app"
	coreerrors "github.com/yogayulanda/go-core/errors"
)

func customErrorHandler(_ *app.App) runtime.ErrorHandlerFunc {
	return func(ctx context.Context, _ *runtime.ServeMux, _ runtime.Marshaler, w http.ResponseWriter, r *http.Request, err error) {
		code := "CORE-9000"
		var details []coreerrors.Detail
		var appErr *coreerrors.AppError
		if errors.As(err, &appErr) {
			def, _ := coreerrors.Resolve(appErr.Code)
			code = def.Code
			if code == "CORE-1000" {
				details = appErr.Details
			}
		} else if info, ok := coreerrors.InfoFromGRPC(err); ok {
			code = info.Code
			if code == "CORE-1000" {
				details = coreerrors.DetailsFromGRPC(err)
			}
		}
		writeGatewayError(ctx, w, r.Header.Get(requestIDHeader), code, details)
	}
}

func writeGatewayError(ctx context.Context, w http.ResponseWriter, requestID, code string, details []coreerrors.Detail) {
	def, registered := coreerrors.Lookup(code)
	if !registered {
		def, _ = coreerrors.Lookup("CORE-9000")
	}
	status := string(def.Class)
	if def.Class == coreerrors.ClassUnknown {
		status = string(coreerrors.ClassPending)
	}
	body := gatewayEnvelope{
		Status:       status,
		ResponseCode: def.Code,
		Message:      coreerrors.Message(ctx, def.Code),
		RequestID:    requestID,
		Timestamp:    time.Now().UTC().Format("2006-01-02T15:04:05.000Z"),
	}
	if def.Code == "CORE-1000" {
		body.Details = details
	}
	w.Header().Set("Content-Type", "application/json")
	if requestID != "" {
		w.Header().Set(requestIDHeader, requestID)
	}
	w.WriteHeader(def.HTTPStatus)
	_ = json.NewEncoder(w).Encode(body)
}
