package gateway

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/yogayulanda/go-core/app"
	"github.com/yogayulanda/go-core/config"
)

func TestHTTPMetricsUseMatchedTemplateAndBoundUnmatchedRoutes(t *testing.T) {
	const serviceName = "route-template-metrics-test"
	application, err := app.New(context.Background(), &config.Config{
		App:       config.AppConfig{ServiceName: serviceName, Environment: "dev", LogLevel: "error", ShutdownTimeout: time.Second},
		Databases: map[string]config.DBConfig{},
	})
	if err != nil {
		t.Fatal(err)
	}
	mux := runtime.NewServeMux(runtime.WithMiddlewares(captureMatchedRoute))
	if err := mux.HandlePath(http.MethodGet, "/v1/x/{id}", func(w http.ResponseWriter, _ *http.Request, _ map[string]string) {
		w.WriteHeader(http.StatusOK)
	}); err != nil {
		t.Fatal(err)
	}
	handler := withHTTPMetrics(application, mux)
	for _, id := range []string{"GTID-1", "GTID-2"} {
		request := httptest.NewRequest(http.MethodGet, "/v1/x/"+id, nil)
		handler.ServeHTTP(httptest.NewRecorder(), request)
	}
	metrics := application.Metrics()
	if got := testutil.ToFloat64(metrics.HTTPRequestTotal.WithLabelValues(serviceName, http.MethodGet, "/v1/x/{id=*}", "200")); got != 2 {
		t.Fatalf("expected both matched IDs to share route template, got count %v", got)
	}
	for index := 0; index < 1000; index++ {
		request := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/random/%d/unmatched", index), nil)
		handler.ServeHTTP(httptest.NewRecorder(), request)
	}
	if got := testutil.ToFloat64(metrics.HTTPRequestTotal.WithLabelValues(serviceName, http.MethodGet, "unmatched", "404")); got != 1000 {
		t.Fatalf("expected unmatched paths to share one bounded label, got count %v", got)
	}
}
