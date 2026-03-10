package logger

import "testing"

func TestParseLevel_ValidAndInvalid(t *testing.T) {
	cases := []string{"debug", "info", "warn", "error"}
	for _, level := range cases {
		if _, err := parseLevel(level); err != nil {
			t.Fatalf("expected valid level %s, got err: %v", level, err)
		}
	}

	lvl, err := parseLevel("invalid")
	if err == nil {
		t.Fatalf("expected error for invalid level")
	}
	if lvl.String() != "info" {
		t.Fatalf("expected fallback info level, got %s", lvl.String())
	}
}

func TestIsDevEnvironment(t *testing.T) {
	if !isDevEnvironment("dev") {
		t.Fatalf("dev must be true")
	}
	if !isDevEnvironment("local") {
		t.Fatalf("local must be true")
	}
	if isDevEnvironment("prod") {
		t.Fatalf("prod must be false")
	}
}
