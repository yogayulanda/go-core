package errors

import "strings"

type AppError struct {
	Code     Code
	Message  string
	Category Category
	Details  []Detail
	Err      error // internal error (not exposed)
}

type Detail struct {
	Field  string `json:"field"`
	Reason string `json:"reason"`
}

type Category string

const (
	CategoryValidation Category = "validation"
	CategoryAuth       Category = "auth"
	CategoryBusiness   Category = "business"
	CategoryPartner    Category = "partner"
	CategoryInternal   Category = "internal"
)

func (e *AppError) Error() string {
	return string(e.Code) + ": " + e.Message
}

func New(code Code, message string) *AppError {
	return &AppError{
		Code:     code,
		Message:  safeMessage(code, message),
		Category: defaultCategory(code),
	}
}

func Wrap(code Code, message string, err error) *AppError {
	return &AppError{
		Code:     code,
		Message:  safeMessage(code, message),
		Category: defaultCategory(code),
		Err:      err,
	}
}

func Validation(message string, details ...Detail) *AppError {
	return &AppError{
		Code:     CodeInvalidRequest,
		Message:  safeMessage(CodeInvalidRequest, message),
		Category: CategoryValidation,
		Details:  details,
	}
}

func safeMessage(code Code, message string) string {
	m := strings.TrimSpace(message)
	if m == "" {
		return defaultMessage(code)
	}
	return m
}

func defaultMessage(code Code) string {
	switch code {
	case CodeInvalidRequest:
		return "invalid request"
	case CodeUnauthorized:
		return "unauthorized request"
	case CodeForbidden:
		return "forbidden request"
	case CodeNotFound:
		return "resource not found"
	case CodeSessionExpired:
		return "session expired"
	case CodeServiceUnavailable:
		return "temporary service issue"
	default:
		return "internal server error"
	}
}

func defaultCategory(code Code) Category {
	switch code {
	case CodeInvalidRequest:
		return CategoryValidation
	case CodeUnauthorized, CodeForbidden, CodeSessionExpired:
		return CategoryAuth
	case CodeNotFound:
		return CategoryBusiness
	case CodeServiceUnavailable:
		return CategoryPartner
	default:
		return CategoryInternal
	}
}
