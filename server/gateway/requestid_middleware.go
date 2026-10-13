package gateway

import (
	"bytes"
	"errors"
	"io"
	"net"
	"net/http"
	"strings"

	"github.com/yogayulanda/go-core/app"
	"github.com/yogayulanda/go-core/logger"
	"github.com/yogayulanda/go-core/observability"
)

const requestIDHeader = "x-request-id"

const (
	clientIPHeader          = "X-Core-Client-IP"
	userAgentHeader         = "X-Core-User-Agent"
	requestIDReplacedHeader = "X-Core-Request-ID-Replaced"
)

func withHTTPRequestIdentity(application *app.App, trustedProxyHops int, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := strings.TrimSpace(r.Header.Get(requestIDHeader))
		replaced := !observability.IsValidRequestID(requestID)
		if replaced {
			requestID = observability.GenerateRequestID()
		}

		ctx := observability.EnsureFieldBag(r.Context())
		ctx = observability.WithRequestID(ctx, requestID)
		if replaced {
			observability.MarkRequestIDReplaced(ctx)
		}
		clientIP := resolveClientIP(r.RemoteAddr, r.Header.Get("X-Forwarded-For"), trustedProxyHops)
		_ = observability.SetClientIP(ctx, clientIP)
		_ = observability.SetUserAgent(ctx, r.UserAgent())
		req := r.WithContext(ctx)
		req.Header.Set(requestIDHeader, requestID)
		req.Header.Set(clientIPHeader, clientIP)
		req.Header.Set(userAgentHeader, r.UserAgent())
		if replaced {
			req.Header.Set(requestIDReplacedHeader, "true")
		}
		w.Header().Set(requestIDHeader, requestID)
		finish := application.TrackInFlight(ctx, r.Method+" "+r.URL.Path)
		defer finish()

		next.ServeHTTP(w, req)
	})
}

func withRequestBodyLimit(application *app.App, maxBodyBytes int64, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if maxBodyBytes <= 0 || r.Body == nil {
			next.ServeHTTP(w, r)
			return
		}
		limitedBody := http.MaxBytesReader(w, r.Body, maxBodyBytes)
		body, err := io.ReadAll(limitedBody)
		_ = r.Body.Close()
		if err != nil {
			var maxBytesError *http.MaxBytesError
			if errors.As(err, &maxBytesError) {
				application.Logger().LogService(r.Context(), logger.ServiceLog{LogType: "access", Class: "FAILED", Method: r.Method, Route: r.URL.Path, StatusCode: http.StatusRequestEntityTooLarge, ResponseCode: "CORE-1005"})
				writeGatewayError(r.Context(), w, observability.GetRequestID(r.Context()), "CORE-1005", nil)
				return
			}
			application.Logger().LogService(r.Context(), logger.ServiceLog{LogType: "access", Class: "ERROR", Method: r.Method, Route: r.URL.Path, StatusCode: http.StatusInternalServerError, ResponseCode: "CORE-9000"})
			writeGatewayError(r.Context(), w, observability.GetRequestID(r.Context()), "CORE-9000", nil)
			return
		}
		r.Body = io.NopCloser(bytes.NewReader(body))
		next.ServeHTTP(w, r)
	})
}

func resolveClientIP(remoteAddress, forwardedFor string, trustedProxyHops int) string {
	connectionIP := remoteAddress
	if host, _, err := net.SplitHostPort(remoteAddress); err == nil {
		connectionIP = host
	}
	if trustedProxyHops <= 0 {
		return connectionIP
	}
	parts := strings.Split(forwardedFor, ",")
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	if len(parts) < trustedProxyHops+1 {
		return connectionIP
	}
	clientIP := parts[len(parts)-trustedProxyHops-1]
	if net.ParseIP(clientIP) == nil {
		return connectionIP
	}
	return clientIP
}
