package errors

type Code string

const (
	// Business errors
	CodeInvalidRequest Code = "INVALID_REQUEST"
	CodeUnauthorized   Code = "UNAUTHORIZED"
	CodeForbidden      Code = "FORBIDDEN"
	CodeNotFound       Code = "NOT_FOUND"
	CodeSessionExpired Code = "SESSION_EXPIRED"

	// System errors
	CodeInternal           Code = "INTERNAL_ERROR"
	CodeServiceUnavailable Code = "SERVICE_UNAVAILABLE"
)
