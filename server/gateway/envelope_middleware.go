package gateway

import (
	"encoding/json"
	"net/http"
	"time"

	"go.opentelemetry.io/otel/trace"

	"github.com/yogayulanda/go-core/observability"
)

type successResponse struct {
	Success       bool            `json:"success"`
	TraceID       string          `json:"trace_id,omitempty"`
	TransactionID string          `json:"transaction_id,omitempty"`
	Timestamp     string          `json:"timestamp"`
	Data          json.RawMessage `json:"data"`
}

type responseBuffer struct {
	http.ResponseWriter
	status      int
	body        []byte
	wroteHeader bool
}

func (r *responseBuffer) WriteHeader(status int) {
	if r.wroteHeader {
		return
	}
	r.status = status
	r.wroteHeader = true
}

func (r *responseBuffer) Write(b []byte) (int, error) {
	if !r.wroteHeader {
		r.WriteHeader(http.StatusOK)
	}
	r.body = append(r.body, b...)
	return len(b), nil
}

func withSuccessEnvelope(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Do not wrap health/ready endpoints
		path := r.URL.Path
		if path == "/health" || path == "/ready" || path == "/version" || path == "/metrics" {
			next.ServeHTTP(w, r)
			return
		}

		buf := &responseBuffer{
			ResponseWriter: w,
			status:         http.StatusOK,
		}

		next.ServeHTTP(buf, r)

		// Only wrap 2xx JSON responses. Errors (4xx/5xx) are already wrapped by customErrorHandler.
		if buf.status >= 200 && buf.status < 300 {
			// Extract context fields
			ctx := r.Context()
			traceID := observability.GetRequestID(ctx)
			if span := trace.SpanFromContext(ctx); span != nil && span.SpanContext().IsValid() {
				traceID = span.SpanContext().TraceID().String()
			}
			txID := observability.GetTransactionID(ctx)

			// Parse data
			var data json.RawMessage
			if len(buf.body) > 0 {
				data = json.RawMessage(buf.body)
			} else {
				data = json.RawMessage("{}") // Default empty object if no body
			}

			wrapped := successResponse{
				Success:       true,
				TraceID:       traceID,
				TransactionID: txID,
				Timestamp:     time.Now().UTC().Format(time.RFC3339),
				Data:          data,
			}

			// Copy original headers
			for k, v := range buf.Header() {
				w.Header()[k] = v
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(buf.status)
			_ = json.NewEncoder(w).Encode(wrapped)
			return
		}

		// For non-2xx or non-JSON, write as-is
		for k, v := range buf.Header() {
			w.Header()[k] = v
		}
		w.WriteHeader(buf.status)
		_, _ = w.Write(buf.body)
	})
}
