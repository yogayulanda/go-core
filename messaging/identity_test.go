package messaging

import (
	"context"
	"testing"
	"time"

	"github.com/yogayulanda/go-core/observability"
)

func TestKafkaIdentityHeadersPropagateOnlyRegisteredLabels(t *testing.T) {
	if err := observability.RegisterPropagatedLabel("messaging_test_region"); err != nil {
		t.Fatal(err)
	}
	ctx := observability.EnsureFieldBag(context.Background())
	ctx = observability.WithRequestID(ctx, "message-req-1")
	if err := observability.SetUserID(ctx, "user-1"); err != nil {
		t.Fatal(err)
	}
	if err := observability.SetLabel(ctx, "messaging_test_region", "north"); err != nil {
		t.Fatal(err)
	}
	if err := observability.SetLabel(ctx, "not_propagated", "private"); err != nil {
		t.Fatal(err)
	}
	message := withOutgoingIdentity(ctx, Message{Headers: map[string]string{"custom": "kept"}})
	if message.Headers["x-request-id"] != "message-req-1" || message.Headers["x-user-id"] != "user-1" || message.Headers["x-label-messaging_test_region"] != "north" {
		t.Fatalf("identity headers missing: %#v", message.Headers)
	}
	if _, exists := message.Headers["x-label-not_propagated"]; exists || message.Headers["custom"] != "kept" {
		t.Fatalf("unexpected header propagation: %#v", message.Headers)
	}
}

func TestKafkaConsumerRestoresIdentityAndCloudEvents(t *testing.T) {
	if err := observability.RegisterPropagatedLabel("messaging_test_client"); err != nil {
		t.Fatal(err)
	}
	ctx := restoreIncomingIdentity(context.Background(), map[string]string{
		"x-request-id":                  "consumer-req-1",
		"x-user-id":                     "user-2",
		"x-label-messaging_test_client": "service-a",
		"x-label-unregistered":          "ignored",
		"ce_specversion":                "1.0",
		"ce_id":                         "event-1",
		"ce_type":                       "account.changed.v1",
		"ce_source":                     "/accounts",
		"ce_time":                       "2026-10-14T00:00:00Z",
		"ce_subject":                    "account-1",
		"ce_sequence":                   "7",
	})
	if observability.GetRequestID(ctx) != "consumer-req-1" || observability.GetUserID(ctx) != "user-2" || observability.Labels(ctx)["messaging_test_client"] != "service-a" {
		t.Fatalf("identity was not restored: request=%q user=%q labels=%#v", observability.GetRequestID(ctx), observability.GetUserID(ctx), observability.Labels(ctx))
	}
	if _, exists := observability.Labels(ctx)["unregistered"]; exists {
		t.Fatal("unregistered label was restored")
	}
	event, ok := CloudEventsFromContext(ctx)
	if !ok || event.ID != "event-1" || event.Sequence != "7" || !event.Time.Equal(time.Date(2026, 10, 14, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("CloudEvents attributes missing or malformed: %+v", event)
	}
	generated := restoreIncomingIdentity(context.Background(), map[string]string{})
	if !observability.IsValidRequestID(observability.GetRequestID(generated)) {
		t.Fatalf("missing request ID was not generated: %q", observability.GetRequestID(generated))
	}
}

func TestPartitionWorkerKeepsPartitionsDeterministic(t *testing.T) {
	if got := partitionWorker(3, 4); got != 3 {
		t.Fatalf("partition 3 was routed to worker %d, want 3", got)
	}
	if got := partitionWorker(4, 4); got != 0 {
		t.Fatalf("partition 4 was routed to worker %d, want 0", got)
	}
	if partitionWorker(3, 4) == partitionWorker(4, 4) {
		t.Fatal("partition routing is not deterministic across workers")
	}
}
