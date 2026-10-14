package gateway

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/yogayulanda/go-core/app"
	"github.com/yogayulanda/go-core/config"
)

func TestRawRoutesPreserveResponsesThroughGateway(t *testing.T) {
	gateway := newRawTestGateway(t, 1024, func(_ context.Context, mux *runtime.ServeMux) error {
		routes := []struct {
			path    string
			handler runtime.HandlerFunc
		}{
			{"raw-json", func(w http.ResponseWriter, _ *http.Request, _ map[string]string) {
				w.Header().Set("Content-Type", "application/json")
				w.Header().Set("X-Partner-Ref", "partner-123")
				_, _ = io.WriteString(w, `{"rc":"00","x":1}`)
			}},
			{"raw-html", func(w http.ResponseWriter, _ *http.Request, _ map[string]string) {
				w.Header().Set("Content-Type", "text/html")
				_, _ = io.WriteString(w, "<html><body>partner</body></html>")
			}},
			{"raw-504", func(w http.ResponseWriter, _ *http.Request, _ map[string]string) {
				w.Header().Set("Content-Type", "text/html")
				w.WriteHeader(http.StatusGatewayTimeout)
				_, _ = io.WriteString(w, "<html>timeout</html>")
			}},
			{"raw-response-code", func(w http.ResponseWriter, _ *http.Request, _ map[string]string) {
				w.Header().Set("X-Response-Code", "PARTNER-00")
				_, _ = io.WriteString(w, `{"rc":"00"}`)
			}},
			{"normal", func(w http.ResponseWriter, _ *http.Request, _ map[string]string) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = io.WriteString(w, `{"ok":true}`)
			}},
		}
		for _, route := range routes {
			handler := route.handler
			if route.path != "normal" {
				handler = Raw(handler)
			}
			if err := mux.HandlePath(http.MethodGet, "/"+route.path, handler); err != nil {
				return err
			}
		}
		if err := mux.HandlePath(http.MethodGet, "/raw-http", func(w http.ResponseWriter, r *http.Request, _ map[string]string) {
			RawHTTP(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "text/plain")
				_, _ = io.WriteString(w, "raw http")
			})).ServeHTTP(w, r)
		}); err != nil {
			return err
		}
		if err := mux.HandlePath(http.MethodPost, "/raw-body-limit", Raw(func(w http.ResponseWriter, _ *http.Request, _ map[string]string) {
			_, _ = io.WriteString(w, "body accepted")
		})); err != nil {
			return err
		}
		return mux.HandlePath(http.MethodPost, "/raw-panic", Raw(func(http.ResponseWriter, *http.Request, map[string]string) {
			panic("raw handler panic")
		}))
	})

	tests := []struct {
		path        string
		status      int
		contentType string
		partnerRef  string
		response    string
	}{
		{"raw-json", http.StatusOK, "application/json", "partner-123", `{"rc":"00","x":1}`},
		{"raw-html", http.StatusOK, "text/html", "", "<html><body>partner</body></html>"},
		{"raw-504", http.StatusGatewayTimeout, "text/html", "", "<html>timeout</html>"},
		{"raw-response-code", http.StatusOK, "", "", `{"rc":"00"}`},
		{"raw-http", http.StatusOK, "text/plain", "", "raw http"},
	}
	for _, test := range tests {
		t.Run(test.path, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodGet, "/"+test.path, nil)
			request.Header.Set("X-Request-ID", "raw-request-"+test.path)
			gateway.server.Handler.ServeHTTP(recorder, request)
			if recorder.Code != test.status || recorder.Header().Get("Content-Type") != test.contentType || recorder.Header().Get("X-Partner-Ref") != test.partnerRef || recorder.Body.String() != test.response {
				t.Fatalf("raw response changed: status=%d headers=%v body=%q", recorder.Code, recorder.Header(), recorder.Body.String())
			}
			if recorder.Header().Get("X-Request-ID") != "raw-request-"+test.path {
				t.Fatalf("raw response request ID header changed: %v", recorder.Header())
			}
			if test.path == "raw-response-code" && recorder.Header().Get("X-Response-Code") != "PARTNER-00" {
				t.Fatalf("raw response-code header was stripped: %v", recorder.Header())
			}
		})
	}

	normal := httptest.NewRecorder()
	gateway.server.Handler.ServeHTTP(normal, httptest.NewRequest(http.MethodGet, "/normal", nil))
	var envelope gatewayEnvelope
	if err := json.Unmarshal(normal.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("normal route was not enveloped: %v; body=%q", err, normal.Body.String())
	}
	if envelope.ResponseCode != "CORE-0000" || string(envelope.Data) != `{"ok":true}` {
		t.Fatalf("normal envelope changed: %+v", envelope)
	}

	panicked := httptest.NewRecorder()
	gateway.server.Handler.ServeHTTP(panicked, httptest.NewRequest(http.MethodPost, "/raw-panic", nil))
	if panicked.Code != http.StatusInternalServerError {
		t.Fatalf("raw panic status = %d, want 500", panicked.Code)
	}
	if err := json.Unmarshal(panicked.Body.Bytes(), &envelope); err != nil || envelope.ResponseCode != "CORE-9000" {
		t.Fatalf("raw panic did not retain panic envelope: body=%q err=%v", panicked.Body.String(), err)
	}

	oversized := httptest.NewRequest(http.MethodPost, "/raw-body-limit", strings.NewReader(strings.Repeat("x", 2048)))
	limited := httptest.NewRecorder()
	gateway.server.Handler.ServeHTTP(limited, oversized)
	if limited.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("raw oversized request status = %d, want 413", limited.Code)
	}
	if err := json.Unmarshal(limited.Body.Bytes(), &envelope); err != nil || envelope.ResponseCode != "CORE-1005" {
		t.Fatalf("raw oversized request did not retain body-limit error: body=%q err=%v", limited.Body.String(), err)
	}
}

func TestRawRouteAccessLogIncludesRequestID(t *testing.T) {
	previousStdout := os.Stdout
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = writer
	defer func() { os.Stdout = previousStdout }()

	application, err := app.New(context.Background(), &config.Config{
		App:       config.AppConfig{ServiceName: "gateway-raw-access-test", Environment: "dev", LogLevel: "info", ShutdownTimeout: time.Second},
		Databases: map[string]config.DBConfig{},
	})
	if err != nil {
		t.Fatal(err)
	}
	gateway, err := New(application, func(_ context.Context, mux *runtime.ServeMux) error {
		return mux.HandlePath(http.MethodPost, "/raw", Raw(func(w http.ResponseWriter, _ *http.Request, _ map[string]string) {
			_, _ = io.WriteString(w, "private-response-body")
		}))
	})
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodPost, "/raw", strings.NewReader("private-request-body"))
	request.Header.Set("X-Request-ID", "raw-access-request-1")
	request.Header.Set("X-Private-Header", "private-header-value")
	gateway.server.Handler.ServeHTTP(httptest.NewRecorder(), request)
	shutdownCtx, cancel := context.WithTimeout(context.Background(), time.Second)
	_ = application.Lifecycle().Shutdown(shutdownCtx)
	cancel()
	_ = writer.Close()
	os.Stdout = previousStdout
	logBytes, err := io.ReadAll(reader)
	_ = reader.Close()
	if err != nil {
		t.Fatal(err)
	}

	accessLines := 0
	var accessEntry map[string]interface{}
	for _, line := range strings.Split(string(logBytes), "\n") {
		var entry map[string]interface{}
		if json.Unmarshal([]byte(line), &entry) != nil {
			continue
		}
		if entry["app.log_type"] == "access" && entry["labels.request_id"] == "raw-access-request-1" {
			accessLines++
			accessEntry = entry
		}
	}
	if accessLines != 1 {
		t.Fatalf("matching raw access log lines = %d, want exactly 1; logs=%s", accessLines, logBytes)
	}
	for _, field := range []string{"app.route", "app.method", "app.status_code", "app.response_code", "app.class", "app.duration_ms", "labels.request_id", "ecs.version"} {
		if accessEntry[field] == nil {
			t.Errorf("raw access log missing standard field %q: %#v", field, accessEntry)
		}
	}
	for _, privateValue := range []string{"private-request-body", "private-response-body", "private-header-value"} {
		if strings.Contains(string(logBytes), privateValue) {
			t.Errorf("raw access log included payload/header value %q", privateValue)
		}
	}
}

func newRawTestGateway(t *testing.T, maxBodyBytes int64, register func(context.Context, *runtime.ServeMux) error) *Gateway {
	t.Helper()
	application := newTestApp(t, &config.Config{
		App:       config.AppConfig{ServiceName: "gateway-raw-test", Environment: "dev", LogLevel: "error", ShutdownTimeout: time.Second},
		HTTP:      config.HTTPConfig{MaxBodyBytes: maxBodyBytes},
		Databases: map[string]config.DBConfig{},
	})
	gateway, err := New(application, register)
	if err != nil {
		t.Fatalf("create gateway: %v", err)
	}
	return gateway
}
