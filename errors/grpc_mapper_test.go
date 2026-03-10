package errors

import (
	"fmt"
	"testing"

	errdetails "google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestToGRPC_ValidationWithDetails(t *testing.T) {
	appErr := Validation(
		"",
		Detail{Field: "user_id", Reason: "required"},
		Detail{Field: "amount", Reason: "must be > 0"},
	)

	err := ToGRPC(appErr)
	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("expected grpc status error")
	}
	if st.Code() != codes.InvalidArgument {
		t.Fatalf("expected invalid argument, got %v", st.Code())
	}
	if st.Message() != "invalid request" {
		t.Fatalf("unexpected message: %s", st.Message())
	}

	gotCode := CodeFromGRPC(err)
	if gotCode != CodeInvalidRequest {
		t.Fatalf("expected code %s, got %s", CodeInvalidRequest, gotCode)
	}

	gotDetails := DetailsFromGRPC(err)
	if len(gotDetails) != 2 {
		t.Fatalf("expected 2 details, got %d", len(gotDetails))
	}
}

func TestToGRPC_UnknownErrorFallback(t *testing.T) {
	err := ToGRPC(status.Error(codes.DataLoss, "raw internal"))
	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("expected grpc status error")
	}
	if st.Code() != codes.Internal {
		t.Fatalf("expected internal, got %v", st.Code())
	}
	if st.Message() != "internal server error" {
		t.Fatalf("unexpected message: %s", st.Message())
	}
}

func TestCodeFromGRPC_UnknownReasonFallbackToStatusCode(t *testing.T) {
	st := status.New(codes.NotFound, "not found")
	withDetails, err := st.WithDetails(&errdetails.ErrorInfo{
		Reason: "UNKNOWN_CODE_FROM_UPSTREAM",
		Domain: "test",
	})
	if err != nil {
		t.Fatalf("with details failed: %v", err)
	}

	if code := CodeFromGRPC(withDetails.Err()); code != CodeNotFound {
		t.Fatalf("expected fallback code %s, got %s", CodeNotFound, code)
	}
}

func TestToGRPC_WrappedAppError_UsesWrappedCode(t *testing.T) {
	wrapped := fmt.Errorf("outer: %w", Validation("invalid request", Detail{Field: "id", Reason: "required"}))

	err := ToGRPC(wrapped)
	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("expected grpc status error")
	}
	if st.Code() != codes.InvalidArgument {
		t.Fatalf("expected invalid argument, got %v", st.Code())
	}
	if code := CodeFromGRPC(err); code != CodeInvalidRequest {
		t.Fatalf("expected code %s, got %s", CodeInvalidRequest, code)
	}
}
