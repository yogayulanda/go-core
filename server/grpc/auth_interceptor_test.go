package grpc

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/yogayulanda/go-core/config"
	"github.com/yogayulanda/go-core/security"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func TestAuthInterceptor_IncludeMethod_RequiresToken(t *testing.T) {
	verifier, _ := newVerifierAndToken(t, config.InternalJWTConfig{
		Enabled:        true,
		Issuer:         "issuer-test",
		Audience:       "aud-test",
		Leeway:         30 * time.Second,
		IncludeMethods: []string{"/history.v1.HistoryService/CreateTransactionHistory"},
	})

	called, err, _ := invokeAuthInterceptor(
		verifier,
		"/history.v1.HistoryService/CreateTransactionHistory",
		nil,
	)
	if called {
		t.Fatalf("handler must not be called for missing token on protected method")
	}
	assertCode(t, err, codes.Unauthenticated)
}

func TestAuthInterceptor_IncludeMethod_WithValidToken_PassesAndInjectsClaims(t *testing.T) {
	verifier, token := newVerifierAndToken(t, config.InternalJWTConfig{
		Enabled:        true,
		Issuer:         "issuer-test",
		Audience:       "aud-test",
		Leeway:         30 * time.Second,
		IncludeMethods: []string{"/history.v1.HistoryService/CreateTransactionHistory"},
	})

	md := metadata.Pairs("authorization", "Bearer "+token)
	called, err, claims := invokeAuthInterceptor(
		verifier,
		"/history.v1.HistoryService/CreateTransactionHistory",
		md,
	)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !called {
		t.Fatalf("handler must be called for valid token")
	}
	if claims == nil || claims.Subject != "user-test-1" {
		t.Fatalf("expected claims injected with subject from token sub")
	}
}

func TestAuthInterceptor_IncludeMethod_SkipUnlistedMethod(t *testing.T) {
	verifier, _ := newVerifierAndToken(t, config.InternalJWTConfig{
		Enabled:        true,
		Issuer:         "issuer-test",
		Audience:       "aud-test",
		Leeway:         30 * time.Second,
		IncludeMethods: []string{"/history.v1.HistoryService/CreateTransactionHistory"},
	})

	called, err, _ := invokeAuthInterceptor(
		verifier,
		"/history.v1.HistoryService/GetUserHistory",
		nil,
	)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !called {
		t.Fatalf("handler should be called for unlisted method when include list is used")
	}
}

func TestAuthInterceptor_ExcludeMethod_BypassOnlyExcluded(t *testing.T) {
	verifier, _ := newVerifierAndToken(t, config.InternalJWTConfig{
		Enabled:        true,
		Issuer:         "issuer-test",
		Audience:       "aud-test",
		Leeway:         30 * time.Second,
		ExcludeMethods: []string{"/grpc.health.v1.Health/Check"},
	})

	called, err, _ := invokeAuthInterceptor(
		verifier,
		"/grpc.health.v1.Health/Check",
		nil,
	)
	if err != nil || !called {
		t.Fatalf("excluded method should bypass auth, err=%v called=%v", err, called)
	}

	called, err, _ = invokeAuthInterceptor(
		verifier,
		"/history.v1.HistoryService/GetUserHistory",
		nil,
	)
	if called {
		t.Fatalf("protected method should not call handler without token")
	}
	assertCode(t, err, codes.Unauthenticated)
}

func invokeAuthInterceptor(
	verifier *security.InternalJWTVerifier,
	fullMethod string,
	md metadata.MD,
) (called bool, err error, claims *security.Claims) {
	ctx := context.Background()
	if md != nil {
		ctx = metadata.NewIncomingContext(ctx, md)
	}

	_, err = authInterceptor(verifier)(
		ctx,
		"struct{}{}",
		&grpc.UnaryServerInfo{FullMethod: fullMethod},
		func(ctx context.Context, req interface{}) (interface{}, error) {
			called = true
			if c, ok := security.FromContext(ctx); ok {
				claims = c
			}
			return "ok", nil
		},
	)
	return called, err, claims
}

func newVerifierAndToken(t *testing.T, cfg config.InternalJWTConfig) (*security.InternalJWTVerifier, string) {
	t.Helper()

	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate rsa key failed: %v", err)
	}

	pubBytes, err := x509.MarshalPKIXPublicKey(&priv.PublicKey)
	if err != nil {
		t.Fatalf("marshal public key failed: %v", err)
	}
	pemKey := string(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pubBytes}))

	cfg.PublicKey = pemKey
	verifier, err := security.NewInternalJWTVerifier(cfg)
	if err != nil {
		t.Fatalf("new verifier failed: %v", err)
	}

	token, err := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{
		"sub": "user-test-1",
		"iss": cfg.Issuer,
		"aud": cfg.Audience,
		"iat": time.Now().Add(-1 * time.Minute).Unix(),
		"nbf": time.Now().Add(-1 * time.Minute).Unix(),
		"exp": time.Now().Add(1 * time.Hour).Unix(),
	}).SignedString(priv)
	if err != nil {
		t.Fatalf("sign token failed: %v", err)
	}

	return verifier, token
}

func assertCode(t *testing.T, err error, expected codes.Code) {
	t.Helper()
	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("expected grpc status error, got: %v", err)
	}
	if st.Code() != expected {
		t.Fatalf("expected grpc code %v, got %v", expected, st.Code())
	}
}
