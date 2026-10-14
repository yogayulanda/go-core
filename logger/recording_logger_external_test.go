package logger_test

import (
	"context"
	"testing"

	"github.com/yogayulanda/go-core/coretest"
	"github.com/yogayulanda/go-core/logger"
	"github.com/yogayulanda/go-core/observability"
)

func TestCoretestRecordingLoggerCapturesGTID(t *testing.T) {
	recorder, err := coretest.NewRecordingLogger()
	if err != nil {
		t.Fatal(err)
	}
	ctx := observability.WithGTID(observability.EnsureFieldBag(context.Background()), "gtid-recorded")
	recorder.LogService(ctx, logger.ServiceLog{Operation: "recorded-operation", Status: "success"})
	entries := recorder.Find("recorded-operation")
	if len(entries) != 1 || entries[0].Fields["labels.gtid"] != "gtid-recorded" {
		t.Fatalf("recorded GTID field missing: %#v", entries)
	}
}
