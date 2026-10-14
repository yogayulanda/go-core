package security

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/yogayulanda/go-core/config"
)

func TestShouldAuthenticate_DefaultEnabled(t *testing.T) {
	v := &InternalJWTVerifier{
		enabled:        true,
		includeMethods: map[string]struct{}{},
		excludeMethods: map[string]struct{}{},
	}

	if !v.ShouldAuthenticate("/history.v1.HistoryService/GetUserHistory") {
		t.Fatalf("expected default auth enforcement for all methods")
	}
}

func TestShouldAuthenticate_IncludeOnly(t *testing.T) {
	v := &InternalJWTVerifier{
		enabled: true,
		includeMethods: map[string]struct{}{
			"/history.v1.HistoryService/CreateTransactionHistory": {},
		},
		excludeMethods: map[string]struct{}{},
	}

	if !v.ShouldAuthenticate("/history.v1.HistoryService/CreateTransactionHistory") {
		t.Fatalf("expected included method to be authenticated")
	}
	if v.ShouldAuthenticate("/history.v1.HistoryService/GetUserHistory") {
		t.Fatalf("expected non-included method to skip auth")
	}
}

func TestShouldAuthenticate_Exclude(t *testing.T) {
	v := &InternalJWTVerifier{
		enabled:        true,
		includeMethods: map[string]struct{}{},
		excludeMethods: map[string]struct{}{
			"/grpc.health.v1.Health/Check": {},
		},
	}

	if v.ShouldAuthenticate("/grpc.health.v1.Health/Check") {
		t.Fatalf("expected excluded method to skip auth")
	}
	if !v.ShouldAuthenticate("/history.v1.HistoryService/GetUserHistory") {
		t.Fatalf("expected non-excluded method to be authenticated")
	}
}

func TestNormalizeLeeway_DefaultAndCustom(t *testing.T) {
	if got := normalizeLeeway(0); got != 30*time.Second {
		t.Fatalf("expected default leeway 30s, got %v", got)
	}
	if got := normalizeLeeway(5 * time.Second); got != 5*time.Second {
		t.Fatalf("expected custom leeway to be preserved, got %v", got)
	}
	if got := normalizeLeeway(-5 * time.Second); got != 0 {
		t.Fatalf("expected negative leeway to normalize to 0, got %v", got)
	}
}

func TestNewInternalJWTVerifier_NormalizeMethodSet(t *testing.T) {
	v, err := NewInternalJWTVerifier(config.InternalJWTConfig{
		Enabled: false,
		IncludeMethods: []string{
			"history.v1.HistoryService/GetUserHistory",
		},
		ExcludeMethods: []string{
			"/grpc.health.v1.Health/Check",
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := v.includeMethods["/history.v1.HistoryService/GetUserHistory"]; !ok {
		t.Fatalf("expected include method to be normalized with leading slash")
	}
	if _, ok := v.excludeMethods["/grpc.health.v1.Health/Check"]; !ok {
		t.Fatalf("expected exclude method to exist")
	}
}

func TestNewInternalJWTVerifier_RequiresExactlyOneKeySource(t *testing.T) {
	tests := []struct {
		name      string
		publicKey string
		jwksURL   string
		wantError string
	}{
		{name: "both sources", publicKey: "not-a-valid-pem", jwksURL: "http://127.0.0.1:1/jwks", wantError: "both are set; remove one"},
		{name: "no source", wantError: "InternalJWT requires exactly one of PublicKey or JWKSEndpoint"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := NewInternalJWTVerifier(config.InternalJWTConfig{
				Enabled:      true,
				PublicKey:    test.publicKey,
				JWKSEndpoint: test.jwksURL,
			})
			if err == nil || !strings.Contains(err.Error(), test.wantError) {
				t.Fatalf("expected constructor error containing %q, got %v", test.wantError, err)
			}
		})
	}
}

func TestNewInternalJWTVerifier_PublicKeyOnly(t *testing.T) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate RSA key: %v", err)
	}
	publicKey, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	if err != nil {
		t.Fatalf("marshal RSA public key: %v", err)
	}
	verifier, err := NewInternalJWTVerifier(config.InternalJWTConfig{
		Enabled:   true,
		PublicKey: string(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: publicKey})),
	})
	if err != nil || verifier == nil {
		t.Fatalf("static-key verifier construction failed: verifier=%v err=%v", verifier, err)
	}
}

func TestInternalJWTVerifier_JWKSRS256(t *testing.T) {
	trustedKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate trusted key: %v", err)
	}
	wrongKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate wrong key: %v", err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"keys": []map[string]string{{
			"kty": "RSA",
			"kid": "trusted-rsa",
			"use": "sig",
			"alg": "RS256",
			"n":   base64.RawURLEncoding.EncodeToString(trustedKey.N.Bytes()),
			"e":   base64.RawURLEncoding.EncodeToString(big.NewInt(int64(trustedKey.PublicKey.E)).Bytes()),
		}}})
	}))
	defer server.Close()

	verifier, err := NewInternalJWTVerifier(config.InternalJWTConfig{
		Enabled:      true,
		JWKSEndpoint: server.URL,
		Leeway:       time.Second,
	})
	if err != nil {
		t.Fatalf("create JWKS verifier: %v", err)
	}

	makeToken := func(key *rsa.PrivateKey) string {
		t.Helper()
		token := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{
			"sub": "user-1",
			"iat": time.Now().Add(-time.Minute).Unix(),
			"nbf": time.Now().Add(-time.Minute).Unix(),
			"exp": time.Now().Add(time.Hour).Unix(),
		})
		token.Header["kid"] = "trusted-rsa"
		encoded, err := token.SignedString(key)
		if err != nil {
			t.Fatalf("sign token: %v", err)
		}
		return encoded
	}

	if claims, err := verifier.Verify(makeToken(trustedKey)); err != nil || claims == nil {
		t.Fatalf("trusted JWKS token rejected: claims=%v err=%v", claims, err)
	}
	if claims, err := verifier.Verify(makeToken(wrongKey)); err == nil || claims != nil {
		t.Fatalf("token signed by unknown key accepted: claims=%v err=%v", claims, err)
	}
}

func TestInternalJWTVerifier_ConfigMetadata(t *testing.T) {
	v := &InternalJWTVerifier{
		enabled:  true,
		issuer:   "issuer",
		audience: "aud",
		leeway:   15 * time.Second,
		includeMethods: map[string]struct{}{
			"/svc/Method": {},
		},
	}

	meta := v.ConfigMetadata()
	if meta["auth_mode"] != "jwt" {
		t.Fatalf("expected jwt auth mode, got %v", meta["auth_mode"])
	}
	if meta["policy_mode"] != "include" {
		t.Fatalf("expected include policy mode, got %v", meta["policy_mode"])
	}
	if meta["include_method_count"] != 1 {
		t.Fatalf("expected include count 1, got %v", meta["include_method_count"])
	}
}
