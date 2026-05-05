package errors

type Finality string

const (
	FinalityBusiness                Finality = "Business"
	FinalityTechnicalRecoverable    Finality = "Technical Recoverable"
	FinalityTechnicalNonRecoverable Finality = "Technical Non-Recoverable"
	FinalityAmbiguous               Finality = "Ambiguous"
)

type Category string

const (
	CategoryVAL  Category = "VAL"
	CategoryAUTH Category = "AUTH"
	CategorySES  Category = "SES"
	CategorySWI  Category = "SWI"
	CategoryDB   Category = "DB"
	CategoryREC  Category = "REC"
)

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
