package gateway

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/yogayulanda/go-core/config"
	"github.com/yogayulanda/go-core/observability"
)

func TestCORSDisabledUnlessOriginAllowed(t *testing.T) {
	for _, test := range []struct {
		name       string
		allow      []string
		origin     string
		wantOrigin string
	}{
		{name: "disabled", origin: "https://client.example"},
		{name: "allowed", allow: []string{"https://client.example"}, origin: "https://client.example", wantOrigin: "https://client.example"},
		{name: "not-listed", allow: []string{"https://other.example"}, origin: "https://client.example"},
	} {
		t.Run(test.name, func(t *testing.T) {
			handler := withCORS(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) }), test.allow, []string{"Content-Type"})
			req := httptest.NewRequest(http.MethodGet, "/resource", nil)
			req.Header.Set("Origin", test.origin)
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)
			if got := rec.Header().Get("Access-Control-Allow-Origin"); got != test.wantOrigin {
				t.Fatalf("expected allow-origin %q, got %q", test.wantOrigin, got)
			}
			if test.wantOrigin != "" {
				if !strings.Contains(rec.Header().Get("Access-Control-Allow-Headers"), "X-Request-ID") || rec.Header().Get("Access-Control-Expose-Headers") != "X-Request-ID" {
					t.Fatalf("request ID header must be allowed and exposed: %#v", rec.Header())
				}
			}
		})
	}
}

func TestRequestBodyLimitRejectsBeforeHandler(t *testing.T) {
	application := newTestApp(t, &config.Config{
		App:       config.AppConfig{ServiceName: "body-limit-test", Environment: "dev", LogLevel: "error", ShutdownTimeout: 1},
		Databases: map[string]config.DBConfig{},
	})
	called := false
	next := http.HandlerFunc(func(http.ResponseWriter, *http.Request) { called = true })
	handler := withRequestBodyLimit(application, 4, next)
	req := httptest.NewRequest(http.MethodPost, "/resource", strings.NewReader("12345"))
	req = req.WithContext(observability.EnsureFieldBag(req.Context()))
	req = req.WithContext(observability.WithRequestID(req.Context(), "body-limit-1"))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if called {
		t.Fatal("oversized request reached the next handler")
	}
	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("expected HTTP 413, got %d", rec.Code)
	}
	var body gatewayEnvelope
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode gateway envelope: %v", err)
	}
	if body.ResponseCode != "CORE-1005" || body.Status != "FAILED" || body.RequestID != "body-limit-1" {
		t.Fatalf("unexpected body-limit envelope: %+v", body)
	}
}
