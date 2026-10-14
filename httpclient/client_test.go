package httpclient

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/sony/gobreaker/v2"
	"github.com/yogayulanda/go-core/coretest"
	"github.com/yogayulanda/go-core/logger"
	"github.com/yogayulanda/go-core/observability"
	"github.com/yogayulanda/go-core/resilience"
)

func TestClient_Get(t *testing.T) {
	// Mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	}))
	defer server.Close()

	log, _ := logger.New("http-test", "debug")
	client := NewClient(log, WithTimeout(5*time.Second))

	resp, err := client.Get(context.Background(), server.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.StatusCode() != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode())
	}

	if string(resp.Body()) != `{"status":"ok"}` {
		t.Errorf("expected response body, got %s", string(resp.Body()))
	}
}

func TestClient_Retry(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts < 2 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	log, _ := logger.New("http-test", "debug")
	client := NewClient(log, WithRetry(resilience.DefaultRetryOptions()))

	resp, err := client.Get(context.Background(), server.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if attempts != 2 {
		t.Errorf("expected 2 attempts, got %d", attempts)
	}

	if resp.StatusCode() != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode())
	}
}

func TestClient_DefaultDoesNotRetryPOSTAndReturnsReceived503(t *testing.T) {
	var attempts int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()

	log, _ := logger.New("http-test", "error")
	client := NewClient(log)
	resp, err := client.Post(context.Background(), server.URL, map[string]string{"value": "once"})
	if err != nil {
		t.Fatalf("received HTTP status should not be returned as a transport error: %v", err)
	}
	if resp == nil || resp.StatusCode() != http.StatusServiceUnavailable {
		t.Fatalf("expected 503 response, got %#v", resp)
	}
	if attempts != 1 {
		t.Fatalf("expected one POST attempt by default, got %d", attempts)
	}
}

func TestClient_CircuitBreakerDoesNotHideReceivedResponse(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		attempts++
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()

	log, _ := logger.New("http-test", "error")
	cb := resilience.DefaultCircuitBreakerOptions("http-test")
	cb.ReadyToTrip = func(counts gobreaker.Counts) bool { return counts.TotalFailures >= 1 }
	client := NewClient(log, WithCircuitBreaker(cb))
	resp, err := client.Get(context.Background(), server.URL)
	if err != nil || resp == nil || resp.StatusCode() != http.StatusServiceUnavailable {
		t.Fatalf("expected received 503 response, resp=%v err=%v", resp, err)
	}
	resp, err = client.Get(context.Background(), server.URL)
	var transportErr *TransportError
	if resp != nil || !errors.As(err, &transportErr) || transportErr.Phase != PhaseBeforeSend || transportErr.Kind != TransportKindCircuitOpen {
		t.Fatalf("expected before-send circuit-open failure, resp=%v err=%v", resp, err)
	}
	if attempts != 1 {
		t.Fatalf("circuit-open request reached the server: %d attempts", attempts)
	}
}

func TestClient_ConnectionRefusedIsBeforeSend(t *testing.T) {
	address := coretest.RefusingAddr(t)
	log, _ := logger.New("http-test", "error")
	client := NewClient(log, WithoutTracing())
	_, err := client.Get(context.Background(), "http://"+address)
	var transportErr *TransportError
	if !errors.As(err, &transportErr) || transportErr.Phase != PhaseBeforeSend || transportErr.Kind != TransportKindConnection {
		t.Fatalf("expected before-send connection error, got %#v", err)
	}
}

func TestClient_TimeoutAfterHeadersIsAfterSend(t *testing.T) {
	server := coretest.SilentServer(t)
	log, _ := logger.New("http-test", "error")
	client := NewClient(log, WithTimeout(200*time.Millisecond), WithoutTracing())
	_, err := client.Get(context.Background(), server.URL)
	var transportErr *TransportError
	if !errors.As(err, &transportErr) || transportErr.Phase != PhaseAfterSend || transportErr.Kind != TransportKindTimeout {
		t.Fatalf("expected after-send timeout, got %#v", err)
	}
}

func TestClient_ConnectionClosedAfterHeadersIsAfterSend(t *testing.T) {
	server := coretest.CloseAfterHeadersServer(t)
	log, _ := logger.New("http-test", "error")
	client := NewClient(log, WithoutTracing())
	_, err := client.Get(context.Background(), server.URL)
	var transportErr *TransportError
	if !errors.As(err, &transportErr) || transportErr.Phase != PhaseAfterSend {
		t.Fatalf("expected after-send error, got %#v", err)
	}
}

func TestClient_IdentityPropagationAndOneSanitizedLog(t *testing.T) {
	if err := observability.RegisterPropagatedLabel("httpclient_test_label"); err != nil {
		t.Fatal(err)
	}
	var headers []http.Header
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		headers = append(headers, r.Header.Clone())
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()
	ctx := observability.EnsureFieldBag(context.Background())
	ctx = observability.WithRequestID(ctx, "req-http-client")
	_ = observability.SetUserID(ctx, "user-42")
	_ = observability.SetLabel(ctx, "httpclient_test_label", "propagated")
	recorder, err := coretest.NewRecordingLogger()
	if err != nil {
		t.Fatal(err)
	}
	client := NewClient(recorder, WithoutTracing())
	_, err = client.Get(ctx, server.URL+"/resource?token=secret")
	if err != nil {
		t.Fatal(err)
	}
	requestURL := strings.Replace(server.URL, "http://", "http://client-user:client-secret@", 1) + "/resource?token=secret"
	_, err = NewClient(recorder, WithInternalPropagation(), WithoutTracing()).Get(ctx, requestURL)
	if err != nil {
		t.Fatal(err)
	}
	if headers[0].Get("X-Request-ID") != "" || headers[0].Get("X-User-ID") != "" || headers[0].Get("X-Label-httpclient_test_label") != "" {
		t.Fatalf("external request received identity headers: %#v", headers[0])
	}
	if headers[1].Get("X-Request-ID") != "req-http-client" || headers[1].Get("X-User-ID") != "user-42" || headers[1].Get("X-Label-httpclient_test_label") != "propagated" {
		t.Fatalf("internal request missing identity headers: %#v", headers[1])
	}
	entries := recorder.Find("httpclient_call")
	if len(entries) != 2 {
		t.Fatalf("expected one service log per call, got %d", len(entries))
	}
	for _, entry := range entries {
		metadata, _ := entry.Fields["app.metadata"].(map[string]any)
		loggedURL, _ := metadata["url"].(string)
		if strings.Contains(loggedURL, "token=") || strings.Contains(loggedURL, "secret") || loggedURL != server.URL+"/resource" {
			t.Fatalf("request URL was not sanitized: %q", loggedURL)
		}
		if entry.Fields["app.status"] != "success" {
			t.Fatalf("unexpected call status: %q", entry.Fields["app.status"])
		}
	}
}

func TestClient_BodyLoggingRequiresAndUsesMasker(t *testing.T) {
	const secret = "response-secret-value"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(secret))
	}))
	defer server.Close()
	recorder, err := coretest.NewRecordingLogger()
	if err != nil {
		t.Fatal(err)
	}
	masker := func([]byte) []byte { return []byte("[masked]") }
	client := NewClient(recorder, WithBodyLogging(masker), WithoutTracing())
	_, err = client.Post(context.Background(), server.URL, map[string]string{"secret": secret})
	if err != nil {
		t.Fatal(err)
	}
	entries := recorder.Find("httpclient_call")
	if len(entries) != 1 {
		t.Fatalf("expected one HTTP client log, got %d", len(entries))
	}
	metadata, _ := entries[0].Fields["app.metadata"].(map[string]any)
	if metadata["request_body"] != "[masked]" || metadata["response_body"] != "[masked]" {
		t.Fatalf("body mask was not applied: %#v", metadata)
	}
	if strings.Contains(fmt.Sprint(metadata), secret) {
		t.Fatal("raw body secret appeared in logged metadata")
	}
}
