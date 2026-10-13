package coretest

import (
	"context"
	"net/http"
	"testing"

	coreerrors "github.com/yogayulanda/go-core/errors"
	"github.com/yogayulanda/go-core/logger"
	"github.com/yogayulanda/go-core/messaging"
	"github.com/yogayulanda/go-core/observability"
	"google.golang.org/grpc/codes"
)

func TestRecordingLoggerCapturesResolvedMaskingAndContextFields(t *testing.T) {
	recorder, err := NewRecordingLogger(logger.WithConfig(logger.Config{MaskKeys: []string{"password"}}))
	if err != nil {
		t.Fatal(err)
	}
	ctx := observability.EnsureFieldBag(context.Background())
	ctx = observability.WithRequestID(ctx, "req-coretest")
	if err := observability.SetLabel(ctx, "gtid", "gtid-1"); err != nil {
		t.Fatal(err)
	}
	recorder.Info(ctx, "recorded", logger.Field{Key: "password", Value: "secret-value"})
	entries := recorder.Find("recorded")
	if len(entries) != 1 || entries[0].Fields["labels.request_id"] != "req-coretest" || entries[0].Fields["labels.gtid"] != "gtid-1" {
		t.Fatalf("identity fields were not resolved: %#v", entries)
	}
	if entries[0].Fields["password"] == "secret-value" {
		t.Fatalf("sensitive value was not masked: %#v", entries[0].Fields)
	}
	if len(recorder.Fields()) != 1 {
		t.Fatalf("Fields should return the emitted entry fields: %#v", recorder.Fields())
	}
}

func TestFakePublisherRecordsMessageAndReturnsConfiguredFailure(t *testing.T) {
	publisher := NewFakePublisher()
	publisher.Err = context.Canceled
	message := messaging.Message{Topic: "topic", Headers: map[string]string{"x-request-id": "req"}}
	if err := publisher.Publish(context.Background(), message); err != context.Canceled {
		t.Fatalf("expected configured publisher error, got %v", err)
	}
	if len(publisher.Messages) != 1 || publisher.Messages[0].Headers["x-request-id"] != "req" {
		t.Fatalf("message was not recorded: %#v", publisher.Messages)
	}
}

func TestHTTPFixtures(t *testing.T) {
	statusServer := StatusServer(t, http.StatusServiceUnavailable)
	response, err := http.Get(statusServer.URL)
	if err != nil {
		t.Fatal(err)
	}
	_ = response.Body.Close()
	if response.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("unexpected status fixture: %d", response.StatusCode)
	}
	if address := RefusingAddr(t); address == "" {
		t.Fatal("refusing address is empty")
	}
}

func TestWithIsolatedRegistryRestoresGlobalCodes(t *testing.T) {
	const code = "TEST-ISOLATED-1"
	t.Run("isolated", func(t *testing.T) {
		WithIsolatedRegistry(t)
		if err := coreerrors.Register(coreerrors.Def{Code: code, Class: coreerrors.ClassFailed, HTTPStatus: 422, GRPCCode: codes.InvalidArgument, Finality: coreerrors.FinalityBusiness, DefaultMessage: "isolated"}); err != nil {
			t.Fatal(err)
		}
		if _, ok := coreerrors.Lookup(code); !ok {
			t.Fatal("isolated registration missing")
		}
	})
	if _, ok := coreerrors.Lookup(code); ok {
		t.Fatal("isolated registry code leaked past test cleanup")
	}
}
