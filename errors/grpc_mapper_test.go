package errors

import (
	"fmt"
	"testing"

	errdetails "google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestToGRPC_RoundTripForStableCodes(t *testing.T) {
	cases := []struct {
		name     string
		appErr   *AppError
		grpcCode codes.Code
		wantCode Code
	}{
		{
			name:     "invalid_request",
			appErr:   Validation("invalid request", Detail{Field: "user_id", Reason: "required"}),
			grpcCode: codes.InvalidArgument,
			wantCode: CodeInvalidRequest,
		},
		{
			name:     "unauthorized",
			appErr:   New(CodeUnauthorized, ""),
			grpcCode: codes.Unauthenticated,
			wantCode: CodeUnauthorized,
		},
		{
			name:     "session_expired",
			appErr:   New(CodeSessionExpired, ""),
			grpcCode: codes.Unauthenticated,
			wantCode: CodeSessionExpired,
		},
		{
			name:     "forbidden",
			appErr:   New(CodeForbidden, ""),
			grpcCode: codes.PermissionDenied,
			wantCode: CodeForbidden,
		},
		{
			name:     "not_found",
			appErr:   New(CodeNotFound, ""),
			grpcCode: codes.NotFound,
			wantCode: CodeNotFound,
		},
		{
			name:     "service_unavailable",
			appErr:   New(CodeServiceUnavailable, ""),
			grpcCode: codes.Unavailable,
			wantCode: CodeServiceUnavailable,
		},
		{
			name:     "internal_error",
			appErr:   New(CodeInternal, ""),
			grpcCode: codes.Internal,
			wantCode: CodeInternal,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := ToGRPC(tc.appErr)
			st, ok := status.FromError(err)
			if !ok {
				t.Fatalf("expected grpc status error")
			}
			if st.Code() != tc.grpcCode {
				t.Fatalf("expected grpc code %v, got %v", tc.grpcCode, st.Code())
			}
			if code := CodeFromGRPC(err); code != tc.wantCode {
				t.Fatalf("expected core code %s, got %s", tc.wantCode, code)
			}
		})
	}
}

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

func TestToGRPC_NonValidation_DoesNotExposeDetails(t *testing.T) {
	err := ToGRPC(&AppError{
		Code:    CodeForbidden,
		Message: "forbidden request",
		Details: []Detail{{Field: "role", Reason: "missing"}},
	})

	if got := DetailsFromGRPC(err); len(got) != 0 {
		t.Fatalf("expected no grpc details for non-validation error, got %v", got)
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

func TestErrorResponseFromError_DirectAppError(t *testing.T) {
	resp := ErrorResponseFromError(Validation("invalid request", Detail{Field: "user_id", Reason: "required"}), "req-1")

	if resp.Code != string(CodeInvalidRequest) {
		t.Fatalf("unexpected code: %s", resp.Code)
	}
	if resp.Message != "invalid request" {
		t.Fatalf("unexpected message: %s", resp.Message)
	}
	if resp.RequestID != "req-1" {
		t.Fatalf("unexpected request_id: %s", resp.RequestID)
	}
	if len(resp.Details) != 1 {
		t.Fatalf("expected 1 detail, got %d", len(resp.Details))
	}
}

func TestErrorResponseFromError_GRPCInternalWithoutCoreInfo_IsSanitized(t *testing.T) {
	resp := ErrorResponseFromError(status.Error(codes.Internal, "raw database panic"), "req-2")

	if resp.Code != string(CodeInternal) {
		t.Fatalf("unexpected code: %s", resp.Code)
	}
	if resp.Message != defaultMessage(CodeInternal) {
		t.Fatalf("expected sanitized internal message, got %s", resp.Message)
	}
	if len(resp.Details) != 0 {
		t.Fatalf("expected no details, got %v", resp.Details)
	}
}

func TestErrorResponseFromError_GRPCValidationWithCoreInfo_UsesDetails(t *testing.T) {
	err := ToGRPC(Validation("invalid request", Detail{Field: "amount", Reason: "must be > 0"}))
	resp := ErrorResponseFromError(err, "req-3")

	if resp.Code != string(CodeInvalidRequest) {
		t.Fatalf("unexpected code: %s", resp.Code)
	}
	if resp.Message != "invalid request" {
		t.Fatalf("unexpected message: %s", resp.Message)
	}
	if len(resp.Details) != 1 {
		t.Fatalf("expected 1 detail, got %d", len(resp.Details))
	}
}
