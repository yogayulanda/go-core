package errors

type ErrorResponse struct {
	Success       bool     `json:"success"`
	Code          string   `json:"code"`
	Message       string   `json:"message"`
	UserMessage   string   `json:"user_message,omitempty"`
	TraceID       string   `json:"trace_id,omitempty"`
	TransactionID string   `json:"transaction_id,omitempty"`
	Timestamp     string   `json:"timestamp"`
	Details       []Detail `json:"details,omitempty"`
}
