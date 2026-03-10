# Error Handling

Use `errors.AppError` as canonical app error.

Stable codes:
`INVALID_REQUEST`, `UNAUTHORIZED`, `FORBIDDEN`, `NOT_FOUND`, `SESSION_EXPIRED`, `SERVICE_UNAVAILABLE`, `INTERNAL_ERROR`.

Transport:
- gRPC: `errors.ToGRPC(err)`
- Gateway: compact JSON `code,message,request_id,details?`

Rules:
- validation may include `details`
- unknown errors must be sanitized
- internal details stay in logs
