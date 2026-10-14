package errors

import (
	"context"
	"errors"
	"testing"

	errdetails "google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type testMessageProvider struct {
	message string
	ok      bool
}

func (p testMessageProvider) Message(context.Context, string) (string, bool) {
	return p.message, p.ok
}

func TestToGRPC_RegistryMappingAndDetails(t *testing.T) {
	grpcErr := ToGRPC(Validation("technical validation detail", Detail{Field: "amount", Reason: "must be positive"}))
	st, ok := status.FromError(grpcErr)
	if !ok || st.Code() != codes.InvalidArgument {
		t.Fatalf("expected InvalidArgument status, got %v", grpcErr)
	}
	if st.Message() != "invalid request" {
		t.Fatalf("technical message leaked or default message missing: %q", st.Message())
	}
	info, ok := InfoFromGRPC(grpcErr)
	if !ok || info.Code != "CORE-1000" || info.Class != ClassFailed || info.Finality != FinalityBusiness || info.Retryable {
		t.Fatalf("unexpected gRPC error metadata: %+v, %v", info, ok)
	}
	gotDetails := DetailsFromGRPC(grpcErr)
	if len(gotDetails) != 1 || gotDetails[0] != (Detail{Field: "amount", Reason: "must be positive"}) {
		t.Fatalf("unexpected validation details: %+v", gotDetails)
	}
}

func TestToGRPC_UnknownCodeFallsBackWithoutLeakingMessage(t *testing.T) {
	grpcErr := ToGRPC(New("UNREGISTERED-CODE", "secret internal detail"))
	st, ok := status.FromError(grpcErr)
	if !ok || st.Code() != codes.Internal || st.Message() != "internal server error" {
		t.Fatalf("unexpected unknown-code mapping: %v", grpcErr)
	}
	info, ok := InfoFromGRPC(grpcErr)
	if !ok || info.Code != "CORE-9000" || info.Class != ClassError {
		t.Fatalf("unexpected fallback info: %+v, %v", info, ok)
	}
}

func TestToGRPC_NonAppErrorIsSanitized(t *testing.T) {
	grpcErr := ToGRPC(errors.New("raw database password detail"))
	st, ok := status.FromError(grpcErr)
	if !ok || st.Code() != codes.Internal || st.Message() != "internal server error" {
		t.Fatalf("non-AppError was not sanitized: %v", grpcErr)
	}
}

func TestMessageProviderOverrideAndFallback(t *testing.T) {
	previous := registry.messageProvider
	t.Cleanup(func() { SetMessageProvider(previous) })

	def, _ := Lookup("CORE-1000")
	SetMessageProvider(testMessageProvider{message: "user-facing override", ok: true})
	if got := messageFor(context.Background(), "CORE-1000", def); got != "user-facing override" {
		t.Fatalf("provider override not used: %q", got)
	}
	SetMessageProvider(testMessageProvider{message: "", ok: false})
	if got := messageFor(context.Background(), "CORE-1000", def); got != "invalid request" {
		t.Fatalf("default message not used when provider declines: %q", got)
	}
}

func TestInfoFromGRPCRejectsNonCoreInfo(t *testing.T) {
	foreign, err := status.New(codes.NotFound, "not found").WithDetails(&errdetails.ErrorInfo{Reason: "CORE-1004", Domain: "another-service"})
	if err != nil {
		t.Fatalf("attach detail: %v", err)
	}
	if _, ok := InfoFromGRPC(foreign.Err()); ok {
		t.Fatal("foreign ErrorInfo must not be treated as go-core")
	}
}
