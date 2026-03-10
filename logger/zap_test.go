package logger

import (
	"testing"
	"time"
)

func TestResolveLogLocation_DefaultUTC(t *testing.T) {
	loc := resolveLogLocation("")
	if loc.String() != time.UTC.String() {
		t.Fatalf("expected UTC, got %s", loc.String())
	}
}

func TestResolveLogLocation_InvalidFallbackUTC(t *testing.T) {
	loc := resolveLogLocation("Invalid/Timezone")
	if loc.String() != time.UTC.String() {
		t.Fatalf("expected UTC fallback, got %s", loc.String())
	}
}
