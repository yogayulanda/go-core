package gateway

import (
	"net/http"
	"runtime/debug"

	"github.com/yogayulanda/go-core/app"
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
					LogType:         "access",
					Class:           "ERROR",
					Method:          r.Method,
					Route:           r.URL.Path,
					StatusCode:      http.StatusInternalServerError,
					ResponseCode:    "CORE-9000",
					ErrorMessage:    "panic recovered",
					ErrorStackTrace: string(debug.Stack()),
				})

				writeGatewayError(ctx, w, requestID, "CORE-9000", nil)
			}
		}()

		next.ServeHTTP(w, r)
	})
}
