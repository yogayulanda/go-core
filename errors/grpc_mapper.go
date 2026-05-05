package errors

import (
	stderrors "errors"
	"strconv"
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

	domain := appErr.Domain
	if domain == "" {
		domain = errorInfoDomain
	}

	stWithInfo, detailErr := st.WithDetails(&errdetails.ErrorInfo{
		Reason: string(code),
		Domain: domain,
		Metadata: map[string]string{
			"category":     string(appErr.Category),
			"number":       appErr.Number,
			"user_message": appErr.UserMessage,
			"retryable":    strconv.FormatBool(appErr.Retryable),
			"finality":     string(appErr.Finality),
			"core_error":   "true",
		},
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

func ErrorResponseFromError(err error, traceID string, txID string) ErrorResponse {
	codeStr, message, userMessage, details := publicErrorContract(err)
	return ErrorResponse{
		Success:       false,
		Code:          codeStr,
		Message:       message,
		UserMessage:   userMessage,
		TraceID:       traceID,
		TransactionID: txID,
		Details:       details,
	}
}

func publicErrorContract(err error) (string, string, string, []Detail) {
	if err == nil {
		return string(CodeInternal), defaultMessage(CodeInternal), "", nil
	}

	var appErr *AppError
	if stderrors.As(err, &appErr) {
		code := normalizeCode(string(appErr.Code))
		message := safeMessage(code, appErr.Message)
		if code != CodeInvalidRequest {
			return appErr.FormatCode(), message, appErr.UserMessage, nil
		}
		return appErr.FormatCode(), message, appErr.UserMessage, normalizeDetails(appErr.Details)
	}

	st, ok := status.FromError(err)
	if !ok {
		return string(CodeInternal), defaultMessage(CodeInternal), "", nil
	}

	code, domain, category, number, userMessage, isCore := ErrorInfoFromGRPC(err)
	if !isCore {
		return string(code), defaultMessage(code), "", nil
	}

	message := safeMessage(code, st.Message())

	formattedCode := string(code)
	if domain != "" && category != "" && number != "" {
		formattedCode = domain + "-" + category + "-" + number
	} else if domain != errorInfoDomain && domain != "" {
		formattedCode = domain + "-" + string(code)
	}

	if code != CodeInvalidRequest {
		return formattedCode, message, userMessage, nil
	}
	return formattedCode, message, userMessage, normalizeDetails(DetailsFromGRPC(err))
}

func CodeFromGRPC(err error) Code {
	code, _, _, _, _, _ := ErrorInfoFromGRPC(err)
	return code
}

func ErrorInfoFromGRPC(err error) (Code, string, string, string, string, bool) {
	st, ok := status.FromError(err)
	if !ok {
		return CodeInternal, "", "", "", "", false
	}

	for _, d := range st.Details() {
		if info, ok := d.(*errdetails.ErrorInfo); ok && info != nil {
			isCore := info.Domain == errorInfoDomain || info.Metadata["core_error"] == "true"
			if !isCore {
				continue
			}
			code := normalizeCode(info.Reason)
			domain := info.Domain
			if domain == errorInfoDomain {
				domain = ""
			}
			category := info.Metadata["category"]
			number := info.Metadata["number"]
			userMessage := info.Metadata["user_message"]

			if code != CodeInternal || info.Reason == string(CodeInternal) {
				return code, domain, category, number, userMessage, true
			}
		}
	}

	return codeFromGRPCStatus(st.Code()), "", "", "", "", false
}

func RetryableFromGRPC(err error) bool {
	st, ok := status.FromError(err)
	if !ok {
		return false
	}
	for _, d := range st.Details() {
		if info, ok := d.(*errdetails.ErrorInfo); ok && info != nil {
			if retryableStr, ok := info.Metadata["retryable"]; ok {
				retryable, _ := strconv.ParseBool(retryableStr)
				return retryable
			}
		}
	}
	return false
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
	_, _, _, _, _, isCore := ErrorInfoFromGRPC(err)
	return isCore
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
