package logger

import (
	"context"
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

func TestLoggerNew_ImplementsExpandedInterface(t *testing.T) {
	log, err := New("test-service", "info")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	log.LogService(context.Background(), ServiceLog{Operation: "boot", Status: "success"})
	log.LogDB(context.Background(), DBLog{Operation: "connect", DBName: "primary", Status: "success"})
}
