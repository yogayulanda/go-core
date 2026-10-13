package gateway

import (
	"net/http"
	"strings"
)

func withCORS(next http.Handler, allowedOrigins, allowedHeaders []string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := strings.TrimSpace(r.Header.Get("Origin"))
		allowed := false
		for _, candidate := range allowedOrigins {
			if strings.TrimSpace(candidate) == origin && origin != "" {
				allowed = true
				break
			}
		}
		if allowed {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Add("Vary", "Origin")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS, PATCH")
			headers := append([]string(nil), allowedHeaders...)
			if !containsHeader(headers, "X-Request-ID") {
				headers = append(headers, "X-Request-ID")
			}
			w.Header().Set("Access-Control-Allow-Headers", strings.Join(headers, ", "))
			w.Header().Set("Access-Control-Expose-Headers", "X-Request-ID")
		}

		// Handle preflight requests
		if r.Method == http.MethodOptions && allowed {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func containsHeader(headers []string, target string) bool {
	for _, header := range headers {
		if strings.EqualFold(strings.TrimSpace(header), target) {
			return true
		}
	}
	return false
}
