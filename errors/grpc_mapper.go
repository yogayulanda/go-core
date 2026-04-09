package errors

import (
	stderrors "errors"
	"strings"

	errdetails "google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const errorInfoDomain = "go-core"

func ToGRPC(err error) error {
	if err == nil {
		return nil
	}

	var appErr *AppError
	if !stderrors.As(err, &appErr) {
		return status.Error(codes.Internal, defaultMessage(CodeInternal))
	}

	code := normalizeCode(string(appErr.Code))
	message := safeMessage(code, appErr.Message)

	st := status.New(grpcCodeFor(code), message)

	stWithInfo, detailErr := st.WithDetails(&errdetails.ErrorInfo{
		Reason: string(code),
		Domain: errorInfoDomain,
	})
	if detailErr == nil {
		st = stWithInfo
	}

	details := normalizeDetails(appErr.Details)
	if code == CodeInvalidRequest && len(details) > 0 {
		br := &errdetails.BadRequest{
			FieldViolations: make([]*errdetails.BadRequest_FieldViolation, 0, len(details)),
		}
		for _, d := range details {
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

func ErrorResponseFromError(err error, requestID string) ErrorResponse {
	code, message, details := publicErrorContract(err)
	return ErrorResponse{
		Code:      string(code),
		Message:   message,
		RequestID: strings.TrimSpace(requestID),
		Details:   details,
	}
}

func publicErrorContract(err error) (Code, string, []Detail) {
	if err == nil {
		return CodeInternal, defaultMessage(CodeInternal), nil
	}

	var appErr *AppError
	if stderrors.As(err, &appErr) {
		code := normalizeCode(string(appErr.Code))
		message := safeMessage(code, appErr.Message)
		if code != CodeInvalidRequest {
			return code, message, nil
		}
		return code, message, normalizeDetails(appErr.Details)
	}

	st, ok := status.FromError(err)
	if !ok {
		return CodeInternal, defaultMessage(CodeInternal), nil
	}

	code := CodeFromGRPC(err)
	if !hasCoreErrorInfo(err) {
		return code, defaultMessage(code), nil
	}

	message := safeMessage(code, st.Message())
	if code != CodeInvalidRequest {
		return code, message, nil
	}
	return code, message, normalizeDetails(DetailsFromGRPC(err))
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

func hasCoreErrorInfo(err error) bool {
	st, ok := status.FromError(err)
	if !ok {
		return false
	}

	for _, d := range st.Details() {
		info, ok := d.(*errdetails.ErrorInfo)
		if !ok || info == nil {
			continue
		}
		return info.Domain == errorInfoDomain && normalizeCode(info.Reason) != CodeInternal || (info.Domain == errorInfoDomain && info.Reason == string(CodeInternal))
	}

	return false
}

func normalizeDetails(in []Detail) []Detail {
	if len(in) == 0 {
		return nil
	}

	out := make([]Detail, 0, len(in))
	for _, d := range in {
		field := strings.TrimSpace(d.Field)
		reason := strings.TrimSpace(d.Reason)
		if field == "" && reason == "" {
			continue
		}
		out = append(out, Detail{
			Field:  field,
			Reason: reason,
		})
	}
	if len(out) == 0 {
		return nil
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
