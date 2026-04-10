package gateway

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/yogayulanda/go-core/app"
	coreErrors "github.com/yogayulanda/go-core/errors"
	"github.com/yogayulanda/go-core/logger"
	"github.com/yogayulanda/go-core/observability"
)

func withPanicRecovery(application *app.App, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		requestID := observability.GetRequestID(ctx)
		
		defer func() {
			if rec := recover(); rec != nil {
				// Log the panic with stack trace and standard schema
				application.Logger().LogService(ctx, logger.ServiceLog{
					Operation: "http_request",
					Status:    "failed",
					ErrorCode: "panic_recovered",
					Metadata: map[string]interface{}{
						"method": r.Method,
						"path":   r.URL.Path,
						"panic":  rec,
					},
				})

				// Return a standardized 500 error response
				w.Header().Set("Content-Type", "application/json")
				if requestID != "" {
					w.Header().Set("x-request-id", requestID)
				}
				w.WriteHeader(http.StatusInternalServerError)

				errResp := coreErrors.ErrorResponse{
					Code:      string(coreErrors.CodeInternal),
					Message:   fmt.Sprintf("internal server error"),
					RequestID: requestID,
				}
				_ = json.NewEncoder(w).Encode(errResp)
			}
		}()

		next.ServeHTTP(w, r)
	})
}
