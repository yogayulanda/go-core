package logger

import "context"

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
type Logger interface {
	// Technical/application logging
	Info(ctx context.Context, msg string, fields ...Field)
	Error(ctx context.Context, msg string, fields ...Field)
	Debug(ctx context.Context, msg string, fields ...Field)
	Warn(ctx context.Context, msg string, fields ...Field)

	// Compliance/domain event logging
	LogEvent(ctx context.Context, e EventLog)

	// Business transaction monitoring logging
	LogTransaction(ctx context.Context, tx TransactionLog)

	// WithComponent allows logical grouping of logs
	// Example: database, grpc, http, scheduler
	WithComponent(component string) Logger
}

type Field struct {
	Key   string
	Value interface{}
}
