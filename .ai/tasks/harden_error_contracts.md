Status: pending

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
