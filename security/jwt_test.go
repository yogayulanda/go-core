package security

import (
	"testing"
	"time"

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
