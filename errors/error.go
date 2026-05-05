package errors

import "strings"

type AppError struct {
	Code        Code
	Message     string
	UserMessage string
	Category    Category
	Finality    Finality
	Retryable   bool
	Details     []Detail
	Domain      string
	Number      string
	Err         error // internal error (not exposed)
}

type Detail struct {
	Field  string `json:"field"`
	Reason string `json:"reason"`
}

func (e *AppError) FormatCode() string {
	if e.Domain != "" && e.Category != "" && e.Number != "" {
		return e.Domain + "-" + string(e.Category) + "-" + e.Number
	}
	return string(e.Code)
}

func (e *AppError) Error() string {
	return e.FormatCode() + ": " + e.Message
}

func (e *AppError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

func New(code Code, message string) *AppError {
	return &AppError{
		Code:     code,
		Message:  safeMessage(code, message),
		Category: defaultCategory(code),
		Finality: defaultFinality(code),
	}
}

func Wrap(code Code, message string, err error) *AppError {
	return &AppError{
		Code:     code,
		Message:  safeMessage(code, message),
		Category: defaultCategory(code),
		Finality: defaultFinality(code),
		Err:      err,
	}
}

func Validation(message string, details ...Detail) *AppError {
	return &AppError{
		Code:     CodeInvalidRequest,
		Message:  safeMessage(CodeInvalidRequest, message),
		Category: CategoryVAL,
		Finality: FinalityBusiness,
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
		return CategoryVAL
	case CodeUnauthorized, CodeForbidden, CodeSessionExpired:
		return CategoryAUTH
	case CodeNotFound:
		return CategoryDB // or Business depending on context, using DB as fallback
	case CodeServiceUnavailable:
		return CategorySWI
	default:
		return CategoryREC
	}
}

func defaultFinality(code Code) Finality {
	switch code {
	case CodeInvalidRequest, CodeUnauthorized, CodeForbidden, CodeNotFound, CodeSessionExpired:
		return FinalityBusiness
	case CodeServiceUnavailable:
		return FinalityTechnicalRecoverable
	case CodeInternal:
		return FinalityTechnicalNonRecoverable
	default:
		return FinalityAmbiguous
	}
}

// ErrorBuilder provides a fluent API to build dynamic AppErrors
type ErrorBuilder struct {
	err *AppError
}

func Build(domain string, category Category, number string) *ErrorBuilder {
	return &ErrorBuilder{
		err: &AppError{
			Domain:   domain,
			Category: category,
			Number:   number,
		},
	}
}

func (b *ErrorBuilder) Message(msg string) *ErrorBuilder {
	b.err.Message = msg
	return b
}

func (b *ErrorBuilder) UserMessage(msg string) *ErrorBuilder {
	b.err.UserMessage = msg
	return b
}

func (b *ErrorBuilder) Finality(f Finality) *ErrorBuilder {
	b.err.Finality = f
	return b
}

func (b *ErrorBuilder) Retryable(r bool) *ErrorBuilder {
	b.err.Retryable = r
	return b
}

func (b *ErrorBuilder) Code(c Code) *ErrorBuilder {
	b.err.Code = c
	return b
}

func (b *ErrorBuilder) Details(details ...Detail) *ErrorBuilder {
	b.err.Details = append(b.err.Details, details...)
	return b
}

func (b *ErrorBuilder) Err(err error) *ErrorBuilder {
	b.err.Err = err
	return b
}

func (b *ErrorBuilder) Done() *AppError {
	if b.err.Code == "" {
		b.err.Code = CodeInternal
	}
	b.err.Message = safeMessage(b.err.Code, b.err.Message)
	if b.err.Finality == "" {
		b.err.Finality = defaultFinality(b.err.Code)
	}
	return b.err
}
