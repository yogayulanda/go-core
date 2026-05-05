package logger

import (
	"context"
)

// Logger is the main structured application logger.
//
// Used for:
// - technical logging
// - request lifecycle
// - debugging
// - warnings & errors
//
// This logger is NOT for business monitoring or compliance tracking.
// Use EventLog or TransactionLog for those purposes.
// TransactionLog is an intentional platform-standard monitoring contract
// for transaction-oriented services, not a generic requirement for all services.
type Logger interface {
	// Technical/application logging
	Info(ctx context.Context, msg string, fields ...Field)
	Error(ctx context.Context, msg string, fields ...Field)
	Debug(ctx context.Context, msg string, fields ...Field)
	Warn(ctx context.Context, msg string, fields ...Field)

	// Standard structured service-flow logging
	LogService(ctx context.Context, s ServiceLog)

	// Standard structured database logging
	LogDB(ctx context.Context, d DBLog)

	// Compliance/domain event logging
	LogEvent(ctx context.Context, e EventLog)

	// Platform-standard transaction monitoring logging for transaction-oriented services
	LogTransaction(ctx context.Context, tx TransactionLog)

	// WithComponent allows logical grouping of logs
	// Example: database, grpc, http, scheduler
	WithComponent(component string) Logger
}

type Field struct {
	Key   string
	Value interface{}
}

// Standardized field keys for consistent logging
const (
	FieldCustomerID       = "customer_id"
	FieldIdempotencyKey   = "idempotency_key"
	FieldPartnerReference = "partner_reference"
	FieldTransactionID    = "transaction_id"
)
