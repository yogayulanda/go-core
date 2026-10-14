package gateway

import (
	"net/http"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
)

type rawResponseMarkerKey struct{}

type rawResponseMarker struct {
	enabled bool
}

// Raw opts a grpc-gateway route handler out of the success response envelope.
func Raw(handler runtime.HandlerFunc) runtime.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request, pathParams map[string]string) {
		markRawResponse(r)
		handler(w, r, pathParams)
	}
}

// RawHTTP opts an HTTP handler out of the success response envelope.
func RawHTTP(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		markRawResponse(r)
		handler.ServeHTTP(w, r)
	})
}

func markRawResponse(r *http.Request) {
	if marker, ok := r.Context().Value(rawResponseMarkerKey{}).(*rawResponseMarker); ok {
		marker.enabled = true
	}
}
