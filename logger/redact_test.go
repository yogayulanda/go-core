package logger

import (
	"testing"
	"unicode/utf8"
)

func TestMaskStringKeepLastN(t *testing.T) {
	got := maskStringKeepLastN("1234567890", 2)
	if got != "********90" {
		t.Fatalf("unexpected mask result: %s", got)
	}

	got = maskStringKeepLastN("12", 2)
	if got != "**" {
		t.Fatalf("unexpected short mask result: %s", got)
	}
}

func TestIsJWTPartMatchesPreviousPredicate(t *testing.T) {
	for value := 0; value <= 255; value++ {
		candidate := string([]byte{byte(value)})
		if got, want := isJWTPart(candidate), previousIsJWTPart(candidate); got != want {
			t.Errorf("byte 0x%02x: isJWTPart=%t, previous predicate=%t", value, got, want)
		}
	}

	for _, value := range []rune{'é', '中', 'Ω', '\u0301', '😀', utf8.RuneError} {
		candidate := string(value)
		if got, want := isJWTPart(candidate), previousIsJWTPart(candidate); got != want {
			t.Errorf("rune %U: isJWTPart=%t, previous predicate=%t", value, got, want)
		}
	}
}

func previousIsJWTPart(part string) bool {
	if part == "" {
		return false
	}
	for _, r := range part {
		// Keep the historical expression intact as the equivalence-test oracle.
		//nolint:staticcheck // Preserve the exact pre-change predicate.
		if !(r >= 'a' && r <= 'z') && !(r >= 'A' && r <= 'Z') && !(r >= '0' && r <= '9') && r != '-' && r != '_' {
			return false
		}
	}
	return true
}

func TestSanitizeFieldValue_SensitiveKey(t *testing.T) {
	got := sanitizeFieldValue("access_token", "abcd1234")
	s, ok := got.(string)
	if !ok {
		t.Fatalf("expected string")
	}
	if s == "abcd1234" {
		t.Fatalf("expected masked value")
	}
	if s != "******34" {
		t.Fatalf("unexpected masked value: %s", s)
	}
}

func TestSanitizeFieldValue_NestedMap(t *testing.T) {
	in := map[string]interface{}{
		"username": "alice",
		"password": "secret123",
	}

	got := sanitizeFieldValue("metadata", in)
	out, ok := got.(map[string]interface{})
	if !ok {
		t.Fatalf("expected map output")
	}

	if out["username"] != "alice" {
		t.Fatalf("username must be unchanged")
	}
	if out["password"] == "secret123" {
		t.Fatalf("expected password masked")
	}
}
