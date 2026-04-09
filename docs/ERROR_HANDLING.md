# Error Handling

Use `errors.AppError` as canonical app error.

Stable codes:
`INVALID_REQUEST`, `UNAUTHORIZED`, `FORBIDDEN`, `NOT_FOUND`, `SESSION_EXPIRED`, `SERVICE_UNAVAILABLE`, `INTERNAL_ERROR`.

Transport:
- gRPC: `errors.ToGRPC(err)`
- Gateway: compact JSON `code,message,request_id,details?` via the same canonical error mapping rules

Rules:
- validation may include `details`
- unknown errors must be sanitized
- internal details stay in logs
- `SESSION_EXPIRED` remains a stable contract code even though it maps to gRPC `Unauthenticated`
- gateway responses should reuse canonical error mapping, not re-interpret transport errors ad hoc
- only trusted validation errors should surface `details`; non-validation and unknown errors stay compact
