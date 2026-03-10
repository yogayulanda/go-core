package version

import (
	"encoding/json"
	"testing"
)

func TestJSON_ReturnsVersionPayload(t *testing.T) {
	origVersion := Version
	origCommit := Commit
	origBuildDate := BuildDate
	defer func() {
		Version = origVersion
		Commit = origCommit
		BuildDate = origBuildDate
	}()

	Version = "1.2.3"
	Commit = "abc123"
	BuildDate = "2026-03-07T00:00:00Z"

	payload := JSON()
	if payload == "" {
		t.Fatalf("expected json payload")
	}

	var got Info
	if err := json.Unmarshal([]byte(payload), &got); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	if got.Version != Version || got.Commit != Commit || got.BuildDate != BuildDate {
		t.Fatalf("unexpected payload: %+v", got)
	}
}
