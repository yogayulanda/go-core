package config

import "testing"

func TestNormalizeDBAlias(t *testing.T) {
	if got := NormalizeDBAlias(" Transaction_History "); got != "transaction_history" {
		t.Fatalf("unexpected normalized alias: %q", got)
	}
}

func TestDatabaseEnvPrefix(t *testing.T) {
	if got := DatabaseEnvPrefix("transaction_history"); got != "DB_TRANSACTION_HISTORY_" {
		t.Fatalf("unexpected env prefix: %q", got)
	}
}
