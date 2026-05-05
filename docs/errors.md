# Error Taxonomy and Handling

The `go-core` library enforces a strictly structured error system built around bounds-checked categories, explicit finality, and traceable metadata. 

## The `AppError` Contract

All application-facing errors must be represented by `AppError`, which provides:
- **Domain**: Prefix assigned by the owning service boundary (e.g., `TRF`, `AUTH`).
- **Category**: Actionable groupings of technical and business behaviors.
- **Number**: The specific identifier.
- **UserMessage**: Safe, user-facing explanation localized for frontends.
- **Finality**: Indicates whether a caller should attempt to retry.
- **Retryable**: Boolean mapping to Finality.

The generated error code string takes the shape `<DOMAIN>-<CATEGORY>-<NUMBER>` (e.g., `TRF-VAL-001`).

## Categories

- `VAL`: Validation errors (Bad Requests, malformed structures)
- `AUTH`: Authentication and permission errors (Unauthorized, Forbidden)
- `SES`: Session-related errors
- `SWI`: Upstream/Switch partner integration errors
- `DB`: Database or caching data resolution errors
- `REC`: Uncategorized generic technical errors

## Finality Levels

- `Business`: Corrective action is needed by the user (e.g., fix a bad request). Non-retryable.
- `Technical Recoverable`: Temporary downstream issues. Retryable (e.g., network blip).
- `Technical Non-Recoverable`: Terminal technical constraints. Non-retryable (e.g., invalid schema).
- `Ambiguous`: Fallback state when intention is not known.

## gRPC Mapping

The `errors/grpc_mapper.go` maps internal `AppError` payloads to gRPC `status` responses automatically, packing extended metadata like `domain`, `user_message`, and `retryable` into the Protobuf `errdetails.ErrorInfo.Metadata` map. This ensures zero data loss between microservices.

## Gateway Edge Emittance

The edge HTTP Gateway `error_handler.go` intercepts gRPC responses and maps them strictly to a 9-field standard JSON representation, seamlessly hydrating `trace_id` from OpenTelemetry, and `transaction_id` from the observability context.
