package security

import (
	"context"
	"testing"

	"google.golang.org/grpc/metadata"
)

func TestExtractFromMetadata_GenericHeaders(t *testing.T) {
	ctx := metadata.NewIncomingContext(
		context.Background(),
		metadata.Pairs(
			"x-subject", "user-1",
			"x-session-id", "session-1",
			"x-role", "admin",
			"x-claim-tenant", "acme",
		),
	)

	claims := ExtractFromMetadata(ctx)
	if claims == nil {
		t.Fatalf("expected claims")
	}
	if claims.Subject != "user-1" {
		t.Fatalf("expected subject user-1, got %q", claims.Subject)
	}
	if claims.Attributes["tenant"] != "acme" {
		t.Fatalf("expected tenant attribute")
	}
}

func TestExtractMetadataSummary_MetadataModeSignals(t *testing.T) {
	ctx := metadata.NewIncomingContext(
		context.Background(),
		metadata.Pairs(
			"x-subject", "user-1",
			"x-role", "admin",
			"x-claim-tenant", "acme",
			"x-claim-region", "id",
		),
	)

	summary := ExtractMetadataSummary(ctx)
	if summary["metadata_present"] != true {
		t.Fatalf("expected metadata_present true, got %v", summary["metadata_present"])
	}
	if summary["claim_count"] != 2 {
		t.Fatalf("expected claim_count 2, got %v", summary["claim_count"])
	}
}
