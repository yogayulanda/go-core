package gateway

import (
	"net/http"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"

	"github.com/yogayulanda/go-core/observability"
)

func registerMetricsEndpoint(mux *runtime.ServeMux) error {
	return mux.HandlePath("GET", "/metrics", func(w http.ResponseWriter, r *http.Request, _ map[string]string) {
		observability.Handler().ServeHTTP(w, r)
	})
}
