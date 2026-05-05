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
	if err.Category != CategorySWI {
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

func TestFormatCode_DynamicDomain(t *testing.T) {
	err := Build("TRF", CategoryVAL, "001").
		Message("internal msg").
		UserMessage("user friendly").
		Finality(FinalityBusiness).
		Retryable(true).
		Done()

	if err.FormatCode() != "TRF-VAL-001" {
		t.Fatalf("expected TRF-VAL-001, got %s", err.FormatCode())
	}
	if err.Error() != "TRF-VAL-001: internal msg" {
		t.Fatalf("expected TRF-VAL-001: internal msg, got %s", err.Error())
	}
}

func TestFormatCode_Fallback(t *testing.T) {
	err := Build("", CategoryVAL, "001").Done()
	if err.FormatCode() != string(CodeInternal) {
		t.Fatalf("expected CodeInternal, got %s", err.FormatCode())
	}
}
