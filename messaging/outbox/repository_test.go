package outbox

import (
	"strings"
	"testing"
	"time"
)

func TestBuildInsertPendingQuery_MySQL_UsesQuestionPlaceholders(t *testing.T) {
	query, args := buildInsertPendingQuery(
		"mysql",
		"id-1",
		"topic-1",
		[]byte("k"),
		[]byte("p"),
		[]byte("{}"),
		"PENDING",
		time.Unix(0, 0),
	)

	if !strings.Contains(query, "`key`") {
		t.Fatalf("expected mysql key quoting, query: %s", query)
	}
	if !strings.Contains(query, "VALUES (?, ?, ?, ?, ?, ?, ?)") {
		t.Fatalf("expected mysql placeholders, query: %s", query)
	}
	if len(args) != 7 {
		t.Fatalf("expected 7 args, got %d", len(args))
	}
}

func TestBuildInsertPendingQuery_Postgres_UsesDollarPlaceholders(t *testing.T) {
	query, args := buildInsertPendingQuery(
		"postgres",
		"id-1",
		"topic-1",
		[]byte("k"),
		[]byte("p"),
		[]byte("{}"),
		"PENDING",
		time.Unix(0, 0),
	)

	if !strings.Contains(query, `"key"`) {
		t.Fatalf("expected postgres key quoting, query: %s", query)
	}
	if !strings.Contains(query, "VALUES ($1, $2, $3, $4, $5, $6, $7)") {
		t.Fatalf("expected postgres placeholders, query: %s", query)
	}
	if len(args) != 7 {
		t.Fatalf("expected 7 args, got %d", len(args))
	}
}

func TestBuildInsertPendingQuery_SQLServer_UsesPPlaceholders(t *testing.T) {
	query, args := buildInsertPendingQuery(
		"sqlserver",
		"id-1",
		"topic-1",
		[]byte("k"),
		[]byte("p"),
		[]byte("{}"),
		"PENDING",
		time.Unix(0, 0),
	)

	if !strings.Contains(query, "[key]") {
		t.Fatalf("expected sqlserver key quoting, query: %s", query)
	}
	if !strings.Contains(query, "VALUES (@p1, @p2, @p3, @p4, @p5, @p6, @p7)") {
		t.Fatalf("expected sqlserver placeholders, query: %s", query)
	}
	if len(args) != 7 {
		t.Fatalf("expected 7 args, got %d", len(args))
	}
}
