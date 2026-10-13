package gateway

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	coreerrors "github.com/yogayulanda/go-core/errors"
)

func TestSuccessEnvelope_DefaultCodeAndStripsResponseCodeHeader(t *testing.T) {
	handler := withSuccessEnvelope(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Grpc-Metadata-X-Response-Code", "CORE-0000")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"item":"ok"}`))
	}))
	req := httptest.NewRequest(http.MethodGet, "/items", nil)
	req.Header.Set(requestIDHeader, "req-1")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected HTTP 200, got %d", rec.Code)
	}
	if rec.Header().Get("X-Response-Code") != "" || rec.Header().Get("Grpc-Metadata-X-Response-Code") != "" {
		t.Fatalf("response code metadata leaked in headers: %#v", rec.Header())
	}
	var body gatewayEnvelope
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode envelope: %v", err)
	}
	if body.Status != "SUCCESS" || body.ResponseCode != "CORE-0000" || body.RequestID != "req-1" || string(body.Data) != `{"item":"ok"}` {
		t.Fatalf("unexpected success envelope: %+v", body)
	}
}

func TestSuccessEnvelope_PendingCodeUsesRegistry(t *testing.T) {
	const code = "TEST-PENDING-GATEWAY"
	if err := coreerrors.Register(coreerrors.Def{Code: code, Class: coreerrors.ClassPending, HTTPStatus: 200, GRPCCode: 0, Finality: coreerrors.FinalityBusiness, DefaultMessage: "processing"}); err != nil {
		t.Fatalf("register code: %v", err)
	}
	handler := withSuccessEnvelope(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("X-Response-Code", code)
		_, _ = w.Write([]byte(`{"state":"pending"}`))
	}))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/items", nil))
	var body gatewayEnvelope
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode envelope: %v", err)
	}
	if body.Status != "PENDING" || body.ResponseCode != code || body.Message != "processing" {
		t.Fatalf("unexpected pending envelope: %+v", body)
	}
}
