package examples_test

import (
	"context"
	"testing"

	"github.com/yogayulanda/go-core/coretest"
	"github.com/yogayulanda/go-core/logger"
	"github.com/yogayulanda/go-core/observability"
)

func TestRecordingLoggerExampleAssertsGTID(t *testing.T) {
	recorder, err := coretest.NewRecordingLogger()
	if err != nil {
		t.Fatal(err)
	}
	ctx := observability.WithGTID(observability.EnsureFieldBag(context.Background()), "gtid-example-1")
	recorder.LogService(ctx, logger.ServiceLog{Operation: "payment", Status: "success"})
	entries := recorder.Find("payment")
	if len(entries) != 1 || entries[0].Fields["labels.gtid"] != "gtid-example-1" {
		t.Fatalf("expected labels.gtid in recording logger: %#v", entries)
	}
}
