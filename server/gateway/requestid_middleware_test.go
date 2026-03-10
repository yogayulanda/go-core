package gateway

import (
	"net/http"
	"net/http/httptest"
	"testing"

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

	withHTTPRequestID(next).ServeHTTP(rec, req)

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

	withHTTPRequestID(next).ServeHTTP(rec, req)

	if got := rec.Header().Get(requestIDHeader); got != incomingID {
		t.Fatalf("expected response request id to preserve incoming, got %s", got)
	}
	if gotCtxID != incomingID {
		t.Fatalf("expected context request id to preserve incoming, got %s", gotCtxID)
	}
}
