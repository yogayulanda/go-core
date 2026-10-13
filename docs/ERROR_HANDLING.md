# Error Handling

Use `errors.AppError` as the canonical application error. Define codes by registering an `errors.Def`; codes are open strings and built-in codes use the `CORE-` prefix.

Built-in codes include `CORE-0000` (success), `CORE-1000` (validation), `CORE-1001` (unauthenticated), `CORE-1002` (session expired), `CORE-1003` (forbidden), `CORE-1004` (not found), `CORE-9000` (internal), and `CORE-9001` (unavailable).

Transport:
- gRPC: `errors.ToGRPC(err)`
- Gateway: until the response-envelope migration, the HTTP gateway retains its existing error JSON shape and obtains status/message/details from the response-code registry

Rules:
- validation may include `details`
- unknown errors must be sanitized
- internal details stay in logs
- Unregistered codes resolve to `CORE-9000`; the server access-log boundary records the original code once.
- Registry-derived gRPC metadata carries class, finality, and retryability.
- only trusted validation errors should surface `details`; non-validation and unknown errors stay compact
