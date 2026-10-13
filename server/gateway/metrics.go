package gateway

import (
	"net/http"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"

	"github.com/yogayulanda/go-core/observability"
)

func registerMetricsEndpoint(mux *runtime.ServeMux, metrics *observability.Metrics) error {
	return mux.HandlePath("GET", "/metrics", func(w http.ResponseWriter, r *http.Request, _ map[string]string) {
		if metrics == nil || !metrics.Enabled {
			http.NotFound(w, r)
			return
		}
		observability.Handler(metrics.Enabled).ServeHTTP(w, r)
	})
}
