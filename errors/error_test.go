package errors

import "testing"

func TestNew_DefaultFallbacks(t *testing.T) {
	err := New(CodeServiceUnavailable, "")
	if err.Message != "temporary service issue" {
		t.Fatalf("unexpected default message: %s", err.Message)
	}
	if err.Category != CategoryPartner {
		t.Fatalf("unexpected category: %s", err.Category)
	}
}
