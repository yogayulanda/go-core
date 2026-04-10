package gateway

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/yogayulanda/go-core/app"
	"github.com/yogayulanda/go-core/config"
)

func TestSignatureMiddleware_Disabled(t *testing.T) {
	cfg := config.Config{}
	cfg.Auth.Signature.Enabled = false

	cfg.App.ServiceName = "test-service"
	cfg.App.LogLevel = "info"
	application, err := app.New(context.Background(), &cfg)
	if err != nil {
		t.Fatalf("failed to create app: %v", err)
	}

	handler := withSignatureValidation(application, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 OK when signature disabled, got %v", rr.Code)
	}
}

func TestSignatureMiddleware_Enabled_Valid(t *testing.T) {
	masterKey := "super-secret"
	cfg := config.Config{}
	cfg.Auth.Signature.Enabled = true
	cfg.Auth.Signature.MasterKey = masterKey
	cfg.Auth.Signature.HeaderKey = "x-sig"
	cfg.Auth.Signature.TimestampKey = "x-ts"
	cfg.Auth.Signature.MaxTimeDrift = 5 * time.Minute

	cfg.App.ServiceName = "test-service"
	cfg.App.LogLevel = "info"
	application, err := app.New(context.Background(), &cfg)
	if err != nil {
		t.Fatalf("failed to create app: %v", err)
	}

	handler := withSignatureValidation(application, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	body := []byte(`{"hello":"world"}`)
	req := httptest.NewRequest("POST", "/api", bytes.NewBuffer(body))

	ts := strconv.FormatInt(time.Now().Unix(), 10)
	payload := "POST:/api:" + ts + ":" + string(body)

	mac := hmac.New(sha256.New, []byte(masterKey))
	mac.Write([]byte(payload))
	sig := hex.EncodeToString(mac.Sum(nil))

	req.Header.Set("x-ts", ts)
	req.Header.Set("x-sig", sig)

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for valid signature, got %v", rr.Code)
	}
}

func TestSignatureMiddleware_Enabled_Invalid(t *testing.T) {
	masterKey := "super-secret"
	cfg := config.Config{}
	cfg.Auth.Signature.Enabled = true
	cfg.Auth.Signature.MasterKey = masterKey
	cfg.Auth.Signature.HeaderKey = "x-sig"
	cfg.Auth.Signature.TimestampKey = "x-ts"
	cfg.Auth.Signature.MaxTimeDrift = 5 * time.Minute

	cfg.App.ServiceName = "test-service"
	cfg.App.LogLevel = "info"
	application, err := app.New(context.Background(), &cfg)
	if err != nil {
		t.Fatalf("failed to create app: %v", err)
	}

	handler := withSignatureValidation(application, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK) // Should not be reached
	}))

	body := []byte(`{"hello":"world"}`)
	req := httptest.NewRequest("POST", "/api", bytes.NewBuffer(body))

	// Simulate replay attack with very old timestamp
	oldTs := strconv.FormatInt(time.Now().Add(-10*time.Minute).Unix(), 10)

	mac := hmac.New(sha256.New, []byte(masterKey))
	mac.Write([]byte("POST:/api:" + oldTs + ":" + string(body)))
	sig := hex.EncodeToString(mac.Sum(nil))

	req.Header.Set("x-ts", oldTs)
	req.Header.Set("x-sig", sig)

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 Unauthorized for replay attack, got %v", rr.Code)
	}
}
