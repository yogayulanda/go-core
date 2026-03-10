package errors

type ErrorResponse struct {
	Code      string   `json:"code"`
	Message   string   `json:"message"`
	RequestID string   `json:"request_id,omitempty"`
	Details   []Detail `json:"details,omitempty"`
}
