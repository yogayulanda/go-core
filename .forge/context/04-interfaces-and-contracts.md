# Interfaces and Contracts

## When to Read
- Read before changing exported transport behavior, event envelopes, or public error/auth contracts.

## Do Not Use This For
- Dependency ownership and provider notes: `07-integrations-and-dependencies.md`.
- Secret handling and auth policy detail: `08-security-and-access.md`.

## Source of Truth
- Transport-facing contracts and compatibility-relevant technical interfaces exposed by the foundation.

## Current Context
- Consuming services own their protobuf, HTTP route, and business payload schemas; `go-core` owns reusable transport behavior and technical envelopes.

## Confirmed Facts
- gRPC server construction is provided by `server/grpc.New(...)`, which installs recovery, request ID, auth, logging, and metrics unary interceptors plus OpenTelemetry gRPC stats handling.
- The HTTP gateway created by `server/gateway.New(...)` always registers `/health`, `/ready`, `/version`, and `/metrics`, and conditionally registers pprof endpoints when `HTTP.PprofEnabled` is true.
- Gateway responses use a custom error handler and success-envelope middleware; docs describe the compact public error payload as `code`, `message`, `request_id`, and optional `details`.
- `errors.AppError` is the canonical application-visible error contract; stable codes include `INVALID_REQUEST`, `UNAUTHORIZED`, `FORBIDDEN`, `NOT_FOUND`, `SESSION_EXPIRED`, `SERVICE_UNAVAILABLE`, and `INTERNAL_ERROR`.
- Messaging contracts use `messaging.Message` with `Topic`, `Key`, `Payload`, and `Headers`; the documented payload envelope should include `event_id`, `event_type`, `event_version`, and `occurred_at`.

## Assumptions
- None.

## Related Files
- `07-integrations-and-dependencies.md`
- `08-security-and-access.md`
- `09-errors-and-resilience.md`
