package gateway

import coreErrors "github.com/yogayulanda/go-core/errors"

type legacyGatewayError struct {
	Success       bool                `json:"success"`
	Code          string              `json:"code"`
	Message       string              `json:"message"`
	TraceID       string              `json:"trace_id,omitempty"`
	TransactionID string              `json:"transaction_id,omitempty"`
	Timestamp     string              `json:"timestamp"`
	Details       []coreErrors.Detail `json:"details,omitempty"`
}
