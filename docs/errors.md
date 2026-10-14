# Error Registry and Handling

The `errors` package uses an open response-code registry. Register each application code with its class, HTTP status, gRPC status, finality, retryability, and safe default message. Built-in codes use the `CORE-` prefix; unknown codes resolve to `CORE-9000`.

## The `AppError` Contract

`AppError` carries a registry code, technical message, optional validation details, and an internal wrapped error. Create errors with `errors.New`, `errors.Wrap`, or `errors.Validation`; register custom definitions with `errors.Register`. The registry is authoritative for class, finality, retryability, and transport statuses. Supply localized messages through `MessageProvider`; otherwise the registry's safe default message is used.

## Finality Levels

- `Business`: Corrective action is needed by the user (e.g., fix a bad request). Non-retryable.
- `Technical Recoverable`: Temporary downstream issues. Retryable (e.g., network blip).
- `Technical Non-Recoverable`: Terminal technical constraints. Non-retryable (e.g., invalid schema).
- `Ambiguous`: Fallback state when intention is not known.

## Registration and gRPC Mapping

Use `Register` to add definitions; duplicate or invalid definitions fail. `Lookup` is safe for concurrent reads. `ToGRPC` uses the registered gRPC status and attaches `ErrorInfo` with code, class, finality, and retryability. Validation details are transported as `BadRequest` field violations. Unknown errors are sanitized to `CORE-9000`.

## Gateway Status Mapping

The gateway resolves registered codes to their configured HTTP status and emits the current envelope: `status`, `response_code`, `message`, `request_id`, and `timestamp`. Successful responses include `data`; only validation errors include `details`. Unknown codes map to `CORE-9000`. Oversized request bodies use `CORE-1005` / HTTP 413, not the validation code.
