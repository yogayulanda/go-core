package gateway

import (
	"context"
	"net/http"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/yogayulanda/go-core/app"
)

type routeCaptureContextKey struct{}

type routeCapture struct{ pattern atomic.Value }

func captureMatchedRoute(next runtime.HandlerFunc) runtime.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request, pathParams map[string]string) {
		if capture, ok := r.Context().Value(routeCaptureContextKey{}).(*routeCapture); ok {
			if pattern, matched := runtime.HTTPPattern(r.Context()); matched {
				capture.pattern.Store(pattern.String())
			}
		}
		next(w, r, pathParams)
	}
}

func withHTTPMetrics(application *app.App, next http.Handler) http.Handler {
	serviceName := application.Config().App.ServiceName
	metrics := application.Metrics()

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !metrics.Enabled {
			next.ServeHTTP(w, r)
			return
		}
		// Skip self-observation to avoid noisy scrape feedback loops.
		if r.URL.Path == "/metrics" {
			next.ServeHTTP(w, r)
			return
		}

		start := time.Now()
		capture := &routeCapture{}
		r = r.WithContext(context.WithValue(r.Context(), routeCaptureContextKey{}, capture))
		ww := &statusCapturingResponseWriter{ResponseWriter: w, statusCode: http.StatusOK}
		next.ServeHTTP(ww, r)

		route := "unmatched"
		if pattern := capture.pattern.Load(); pattern != nil {
			route, _ = pattern.(string)
		}
		status := strconv.Itoa(ww.statusCode)
		duration := time.Since(start).Seconds()
		serviceStatus := "success"
		if ww.statusCode >= http.StatusBadRequest {
			serviceStatus = "failed"
		}
		operation := "http_request"

		metrics.HTTPRequestTotal.WithLabelValues(
			serviceName,
			r.Method,
			route,
			status,
		).Inc()

		metrics.HTTPRequestDuration.WithLabelValues(
			serviceName,
			r.Method,
			route,
		).Observe(duration)

		metrics.ServiceTotal.WithLabelValues(
			serviceName,
			operation,
			serviceStatus,
		).Inc()

		metrics.ServiceDuration.WithLabelValues(
			serviceName,
			operation,
		).Observe(duration)

	})
}

type statusCapturingResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (w *statusCapturingResponseWriter) WriteHeader(statusCode int) {
	w.statusCode = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}
