package httpclient

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/yogayulanda/go-core/logger"
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
	client := NewClient(log)

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
