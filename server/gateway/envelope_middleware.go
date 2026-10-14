package gateway

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	coreerrors "github.com/yogayulanda/go-core/errors"
	"github.com/yogayulanda/go-core/logger"
	"github.com/yogayulanda/go-core/observability"
	"github.com/yogayulanda/go-core/response"
)

type gatewayEnvelope struct {
	Status       string              `json:"status"`
	ResponseCode string              `json:"response_code"`
	Message      string              `json:"message"`
	RequestID    string              `json:"request_id"`
	Timestamp    string              `json:"timestamp"`
	Data         json.RawMessage     `json:"data,omitempty"`
	Details      []coreerrors.Detail `json:"details,omitempty"`
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
	return withSuccessEnvelopeAndLogger(next, nil)
}

func withSuccessEnvelopeAndLogger(next http.Handler, log logger.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if isOperationalPath(r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}
		raw := &rawResponseMarker{}
		r = r.WithContext(context.WithValue(r.Context(), rawResponseMarkerKey{}, raw))
		buffered := &responseBuffer{ResponseWriter: w, status: http.StatusOK}
		startedAt := time.Now()
		next.ServeHTTP(buffered, r)
		if raw.enabled {
			w.WriteHeader(buffered.status)
			_, _ = w.Write(buffered.body)
			if log != nil {
				class := "SUCCESS"
				if buffered.status >= http.StatusInternalServerError {
					class = "ERROR"
				} else if buffered.status >= http.StatusBadRequest {
					class = "FAILED"
				}
				responseCode := response.CodeFromHTTPHeader(buffered.Header())
				if responseCode == "" {
					responseCode = "CORE-0000"
				}
				log.LogService(r.Context(), logger.ServiceLog{
					LogType:           "access",
					Class:             class,
					Route:             r.URL.Path,
					Method:            r.Method,
					StatusCode:        buffered.status,
					ResponseCode:      responseCode,
					DurationMs:        time.Since(startedAt).Milliseconds(),
					RequestIDReplaced: observability.RequestIDReplaced(r.Context()),
				})
			}
			return
		}

		responseCode := response.CodeFromHTTPHeader(buffered.Header())
		buffered.Header().Del("X-Response-Code")
		buffered.Header().Del("Grpc-Metadata-X-Response-Code")
		if responseCode == "" {
			responseCode = "CORE-0000"
		}
		def, registered := coreerrors.Lookup(responseCode)
		if !registered {
			def, _ = coreerrors.Lookup("CORE-9000")
		}
		if buffered.status < 200 || buffered.status >= 300 {
			copyHeaders(w.Header(), buffered.Header())
			w.WriteHeader(buffered.status)
			_, _ = w.Write(buffered.body)
			return
		}

		data := json.RawMessage(buffered.body)
		if len(data) == 0 {
			data = json.RawMessage("{}")
		}
		status := string(def.Class)
		if def.Class == coreerrors.ClassUnknown {
			status = string(coreerrors.ClassPending)
		}
		wrapped := gatewayEnvelope{
			Status:       status,
			ResponseCode: def.Code,
			Message:      coreerrors.Message(r.Context(), def.Code),
			RequestID:    responseRequestID(r),
			Timestamp:    time.Now().UTC().Format("2006-01-02T15:04:05.000Z"),
			Data:         data,
		}
		copyHeaders(w.Header(), buffered.Header())
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(def.HTTPStatus)
		_ = json.NewEncoder(w).Encode(wrapped)
	})
}

func responseRequestID(r *http.Request) string {
	return r.Header.Get(requestIDHeader)
}

func isOperationalPath(path string) bool {
	return path == "/health" || path == "/ready" || path == "/version" || path == "/metrics" || path == "/debug/pprof" || len(path) > len("/debug/pprof/") && path[:len("/debug/pprof/")] == "/debug/pprof/"
}

func copyHeaders(destination, source http.Header) {
	for key, values := range source {
		destination[key] = append([]string(nil), values...)
	}
}
