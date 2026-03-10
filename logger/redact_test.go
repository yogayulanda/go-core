package logger

import "testing"

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
