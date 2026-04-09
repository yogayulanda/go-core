Status: done

Task: harden error contracts

Goal:
improve consistency between app errors, gRPC mapping, REST responses, and service-facing guidance

Scope Layers:

runtime
tests
docs
ai

Allowed Paths:

errors/
server/gateway/
server/grpc/
docs/
.ai/
README.md

Constraints:

keep external error payloads compact and sanitized
keep internal diagnostic detail in logs
prefer additive improvements over broad error-contract churn

Expected Output:

- tighter REST/gRPC error consistency
- clearer tests for mapping behavior
- docs that reflect the actual service-facing contract

Implemented Notes:

- `errors/` now owns the canonical public error response mapping for direct app errors and gRPC transport errors
- gRPC mapping preserves stable contract codes, including `SESSION_EXPIRED`, through `ErrorInfo.reason`
- gateway error responses now reuse canonical error mapping and sanitize unknown transport errors
- tests cover stable code round-trip, validation detail exposure, and gateway compact response behavior
