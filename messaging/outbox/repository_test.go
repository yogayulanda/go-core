package outbox

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/yogayulanda/go-core/messaging"
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

	if !strings.Contains(query, "msg_key") || !strings.Contains(query, "event_id") {
		t.Fatalf("expected mysql event ID and message key columns, query: %s", query)
	}
	if !strings.Contains(query, "VALUES (?, ?, ?, ?, ?, ?, 0, NULL, ?, ?)") {
		t.Fatalf("expected mysql placeholders, query: %s", query)
	}
	if len(args) != 8 {
		t.Fatalf("expected 8 args, got %d", len(args))
	}
}

func TestPublisherPublishTx_UnsupportedDriver_ReturnsError(t *testing.T) {
	for _, publisher := range []*Publisher{
		NewPublisher(),
		NewPublisher(WithPublisherDriver("postgres")),
		NewPublisher(WithPublisherDriver("unknown")),
	} {
		if err := publisher.PublishTx(context.Background(), nil, messaging.Message{}); err == nil {
			t.Fatal("expected unsupported driver error")
		}
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

	if !strings.Contains(query, "msg_key") || !strings.Contains(query, "event_id") {
		t.Fatalf("expected SQL Server event ID and message key columns, query: %s", query)
	}
	if !strings.Contains(query, "VALUES (@p1, @p2, @p3, @p4, @p5, @p6, 0, NULL, @p7, @p7)") {
		t.Fatalf("expected sqlserver placeholders, query: %s", query)
	}
	if len(args) != 7 {
		t.Fatalf("expected 7 args, got %d", len(args))
	}
}
