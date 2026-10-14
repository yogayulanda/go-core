package errors

import "strings"

type AppError struct {
	Code    string
	Message string
	Details []Detail
	Err     error
}

type Detail struct {
	Field  string `json:"field"`
	Reason string `json:"reason"`
}

func (e *AppError) Error() string {
	return e.Code + ": " + e.Message
}

func (e *AppError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

func New(code string, message string) *AppError {
	return &AppError{Code: code, Message: strings.TrimSpace(message)}
}

func Wrap(code string, message string, err error) *AppError {
	return &AppError{Code: code, Message: strings.TrimSpace(message), Err: err}
}

func Validation(message string, details ...Detail) *AppError {
	return &AppError{
		Code:    "CORE-1000",
		Message: strings.TrimSpace(message),
		Details: details,
	}
}
