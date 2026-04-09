package errors

import (
	"errors"
	"testing"
)

func TestNew_DefaultFallbacks(t *testing.T) {
	err := New(CodeServiceUnavailable, "")
	if err.Message != "temporary service issue" {
		t.Fatalf("unexpected default message: %s", err.Message)
	}
	if err.Category != CategoryPartner {
		t.Fatalf("unexpected category: %s", err.Category)
	}
}

func TestAppError_Unwrap_ReturnsInternalErr(t *testing.T) {
	inner := errors.New("db down")
	appErr := Wrap(CodeServiceUnavailable, "", inner)

	if !errors.Is(appErr, inner) {
		t.Fatalf("expected wrapped internal error to be discoverable")
	}
}
