package errors

import (
	stderrors "errors"

	errdetails "google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func ToGRPC(err error) error {
	if err == nil {
		return nil
	}

	var appErr *AppError
	if !stderrors.As(err, &appErr) {
		return status.Error(codes.Internal, defaultMessage(CodeInternal))
	}

	st := status.New(grpcCodeFor(appErr.Code), appErr.Message)

	stWithInfo, detailErr := st.WithDetails(&errdetails.ErrorInfo{
		Reason: string(appErr.Code),
		Domain: "go-core",
	})
	if detailErr == nil {
		st = stWithInfo
	}

	if len(appErr.Details) > 0 {
		br := &errdetails.BadRequest{
			FieldViolations: make([]*errdetails.BadRequest_FieldViolation, 0, len(appErr.Details)),
		}
		for _, d := range appErr.Details {
			br.FieldViolations = append(br.FieldViolations, &errdetails.BadRequest_FieldViolation{
				Field:       d.Field,
				Description: d.Reason,
			})
		}
		stWithValidation, err := st.WithDetails(br)
		if err == nil {
			st = stWithValidation
		}
	}

	return st.Err()
}

func CodeFromGRPC(err error) Code {
	st, ok := status.FromError(err)
	if !ok {
		return CodeInternal
	}

	for _, d := range st.Details() {
		if info, ok := d.(*errdetails.ErrorInfo); ok && info != nil {
			if code := normalizeCode(info.Reason); code != CodeInternal || info.Reason == string(CodeInternal) {
				return code
			}
		}
	}

	return codeFromGRPCStatus(st.Code())
}

func DetailsFromGRPC(err error) []Detail {
	st, ok := status.FromError(err)
	if !ok {
		return nil
	}

	var out []Detail
	for _, d := range st.Details() {
		br, ok := d.(*errdetails.BadRequest)
		if !ok || br == nil {
			continue
		}
		for _, fv := range br.FieldViolations {
			if fv == nil {
				continue
			}
			out = append(out, Detail{
				Field:  fv.Field,
				Reason: fv.Description,
			})
		}
	}
	return out
}

func grpcCodeFor(code Code) codes.Code {
	switch code {
	case CodeInvalidRequest:
		return codes.InvalidArgument
	case CodeUnauthorized, CodeSessionExpired:
		return codes.Unauthenticated
	case CodeForbidden:
		return codes.PermissionDenied
	case CodeNotFound:
		return codes.NotFound
	case CodeServiceUnavailable:
		return codes.Unavailable
	default:
		return codes.Internal
	}
}

func codeFromGRPCStatus(code codes.Code) Code {
	switch code {
	case codes.InvalidArgument:
		return CodeInvalidRequest
	case codes.Unauthenticated:
		return CodeUnauthorized
	case codes.PermissionDenied:
		return CodeForbidden
	case codes.NotFound:
		return CodeNotFound
	case codes.Unavailable:
		return CodeServiceUnavailable
	default:
		return CodeInternal
	}
}

func normalizeCode(raw string) Code {
	switch Code(raw) {
	case CodeInvalidRequest,
		CodeUnauthorized,
		CodeForbidden,
		CodeNotFound,
		CodeSessionExpired,
		CodeServiceUnavailable,
		CodeInternal:
		return Code(raw)
	default:
		return CodeInternal
	}
}
