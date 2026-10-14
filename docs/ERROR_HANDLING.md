# Error Handling

Use `errors.AppError` as the canonical application error. Define codes by registering an `errors.Def`; codes are open strings and built-in codes use the `CORE-` prefix.

Built-in codes include `CORE-0000` (success), `CORE-1000` (validation), `CORE-1001` (unauthenticated), `CORE-1002` (session expired), `CORE-1003` (forbidden), `CORE-1004` (not found), `CORE-1005` (body too large), `CORE-9000` (internal), and `CORE-9001` (unavailable).

Transport:
- gRPC: `errors.ToGRPC(err)`
- Gateway: registry definitions determine HTTP status and response class. Responses use the gateway envelope.

Envelope fields are `status`, `response_code`, `message`, `request_id`, and `timestamp`. Successful responses include `data`; validation errors may include `details`. Operational endpoints are not wrapped in the API envelope.

Rules:
- validation may include `details`
- unknown errors must be sanitized
- internal details stay in logs
- Unregistered codes resolve to `CORE-9000`; the server access-log boundary records the original code once.
- Registry-derived gRPC metadata carries class, finality, and retryability.
- only `CORE-1000` validation errors surface field `details`; other failures and unknown errors stay compact and sanitized
- oversized HTTP bodies use `CORE-1005`, HTTP 413, and gRPC `ResourceExhausted`; `CORE-1000` remains HTTP 400 validation
