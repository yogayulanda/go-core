package gateway

import (
	"net/http"
	"strings"

	"github.com/yogayulanda/go-core/observability"
)

const requestIDHeader = "x-request-id"

func withHTTPRequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := strings.TrimSpace(r.Header.Get(requestIDHeader))
		if requestID == "" {
			requestID = observability.GenerateRequestID()
		}

		ctx := observability.WithRequestID(r.Context(), requestID)
		req := r.WithContext(ctx)
		req.Header.Set(requestIDHeader, requestID)
		w.Header().Set(requestIDHeader, requestID)

		next.ServeHTTP(w, req)
	})
}
