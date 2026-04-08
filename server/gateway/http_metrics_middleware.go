package gateway

import (
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/yogayulanda/go-core/app"
	"github.com/yogayulanda/go-core/logger"
)

var uuidSegmentPattern = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[1-5][0-9a-fA-F]{3}-[89abAB][0-9a-fA-F]{3}-[0-9a-fA-F]{12}$`)

func withHTTPMetrics(application *app.App, next http.Handler) http.Handler {
	serviceName := application.Config().App.ServiceName
	metrics := application.Metrics()
	log := application.Logger()

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip self-observation to avoid noisy scrape feedback loops.
		if r.URL.Path == "/metrics" {
			next.ServeHTTP(w, r)
			return
		}

		start := time.Now()
		ww := &statusCapturingResponseWriter{ResponseWriter: w, statusCode: http.StatusOK}
		next.ServeHTTP(ww, r)

		route := normalizeHTTPRoute(r.URL.Path)
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

		log.LogService(r.Context(), logger.ServiceLog{
			Operation:  operation,
			Status:     serviceStatus,
			DurationMs: time.Since(start).Milliseconds(),
			ErrorCode:  httpErrorCode(ww.statusCode),
			Metadata: map[string]interface{}{
				"http_method": r.Method,
				"route":       route,
				"status_code": ww.statusCode,
			},
		})
	})
}

func httpErrorCode(statusCode int) string {
	if statusCode < http.StatusBadRequest {
		return ""
	}
	if statusCode >= http.StatusInternalServerError {
		return "http_server_error"
	}
	return "http_client_error"
}

type statusCapturingResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (w *statusCapturingResponseWriter) WriteHeader(statusCode int) {
	w.statusCode = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}

func normalizeHTTPRoute(path string) string {
	if path == "" || path == "/" {
		return "/"
	}

	rawSegments := strings.Split(path, "/")
	segments := make([]string, 0, len(rawSegments))

	for _, segment := range rawSegments {
		if segment == "" {
			continue
		}

		switch {
		case isNumericSegment(segment):
			segments = append(segments, ":id")
		case isUUIDSegment(segment):
			segments = append(segments, ":id")
		case isLikelyDynamicSegment(segment):
			segments = append(segments, ":param")
		default:
			segments = append(segments, segment)
		}
	}

	if len(segments) == 0 {
		return "/"
	}

	return "/" + strings.Join(segments, "/")
}

func isNumericSegment(v string) bool {
	_, err := strconv.ParseInt(v, 10, 64)
	return err == nil
}

func isUUIDSegment(v string) bool {
	return uuidSegmentPattern.MatchString(v)
}

func isLikelyDynamicSegment(v string) bool {
	if len(v) >= 32 {
		return true
	}

	dashCount := strings.Count(v, "-")
	if dashCount >= 2 && len(v) >= 12 {
		return true
	}

	hasLetter := false
	hasDigit := false
	for _, r := range v {
		if unicode.IsLetter(r) {
			hasLetter = true
		}
		if unicode.IsDigit(r) {
			hasDigit = true
		}
	}

	if hasLetter && hasDigit && len(v) >= 8 {
		return true
	}

	return false
}
