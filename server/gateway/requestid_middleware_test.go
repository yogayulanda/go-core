package gateway

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/yogayulanda/go-core/app"
	"github.com/yogayulanda/go-core/config"
	"github.com/yogayulanda/go-core/observability"
)

func TestWithHTTPRequestID_GenerateWhenMissing(t *testing.T) {
	var gotCtxID string
	var gotHeaderID string

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotCtxID = observability.GetRequestID(r.Context())
		gotHeaderID = r.Header.Get(requestIDHeader)
		w.WriteHeader(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	withHTTPRequestIdentity(requestIDTestApp(t), 0, next).ServeHTTP(rec, req)

	respHeaderID := rec.Header().Get(requestIDHeader)
	if respHeaderID == "" {
		t.Fatalf("expected response request id header")
	}
	if gotCtxID != respHeaderID {
		t.Fatalf("expected context request id to match response header, got ctx=%s header=%s", gotCtxID, respHeaderID)
	}
	if gotHeaderID != respHeaderID {
		t.Fatalf("expected request header propagated, got req=%s header=%s", gotHeaderID, respHeaderID)
	}
}

func TestWithHTTPRequestID_ReuseIncoming(t *testing.T) {
	const incomingID = "req-incoming-123"

	var gotCtxID string
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotCtxID = observability.GetRequestID(r.Context())
		w.WriteHeader(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	req.Header.Set(requestIDHeader, incomingID)
	rec := httptest.NewRecorder()

	withHTTPRequestIdentity(requestIDTestApp(t), 0, next).ServeHTTP(rec, req)

	if got := rec.Header().Get(requestIDHeader); got != incomingID {
		t.Fatalf("expected response request id to preserve incoming, got %s", got)
	}
	if gotCtxID != incomingID {
		t.Fatalf("expected context request id to preserve incoming, got %s", gotCtxID)
	}
}

func TestHTTPRequestIdentity_ReplacesInvalidRequestID(t *testing.T) {
	var replaced bool
	next := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		replaced = observability.RequestIDReplaced(r.Context())
	})
	req := httptest.NewRequest(http.MethodGet, "/resource", nil)
	req.Header.Set(requestIDHeader, "../../etc")
	withHTTPRequestIdentity(requestIDTestApp(t), 0, next).ServeHTTP(httptest.NewRecorder(), req)
	if !replaced {
		t.Fatal("invalid request ID must be marked as replaced")
	}
}

func TestHTTPRequestIdentity_TrustedProxyHops(t *testing.T) {
	tests := []struct {
		name string
		hops int
		want string
	}{
		{name: "trust one proxy", hops: 1, want: "1.1.1.1"},
		{name: "trust no proxies", hops: 0, want: "192.0.2.1"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var gotIP, gotAgent, gotForwardedIP, gotForwardedAgent string
			next := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
				gotIP = observability.GetClientIP(r.Context())
				gotAgent = observability.GetUserAgent(r.Context())
				gotForwardedIP = r.Header.Get(clientIPHeader)
				gotForwardedAgent = r.Header.Get(userAgentHeader)
			})
			req := httptest.NewRequest(http.MethodGet, "/resource", nil)
			req.RemoteAddr = "192.0.2.1:1234"
			req.Header.Set("X-Forwarded-For", "1.1.1.1, 10.0.0.2")
			req.Header.Set("User-Agent", "client/1.2")
			withHTTPRequestIdentity(requestIDTestApp(t), test.hops, next).ServeHTTP(httptest.NewRecorder(), req)
			if gotIP != test.want || gotForwardedIP != test.want {
				t.Fatalf("expected client IP %s, bag=%s header=%s", test.want, gotIP, gotForwardedIP)
			}
			if gotAgent != "client/1.2" || gotForwardedAgent != gotAgent {
				t.Fatalf("user agent not captured and forwarded: bag=%q header=%q", gotAgent, gotForwardedAgent)
			}
		})
	}
}

func requestIDTestApp(t *testing.T) *app.App {
	t.Helper()
	return newTestApp(t, &config.Config{
		App:       config.AppConfig{ServiceName: "request-id-test", Environment: "dev", LogLevel: "error", ShutdownTimeout: time.Second},
		Databases: map[string]config.DBConfig{},
	})
}

func TestResolveClientIP_InsufficientForwardedChainUsesConnection(t *testing.T) {
	if got := resolveClientIP("192.0.2.1:1234", "1.1.1.1", 2); got != "192.0.2.1" {
		t.Fatalf("expected connection address fallback, got %q", got)
	}
}
