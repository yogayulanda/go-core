# Proposed Context Patch: 08 Security And Access

Status: promoted on 2026-10-14 to `.forge/context/08-security-and-access.md`.

## Add confirmed facts
- A non-empty `AUTH_SIGNATURE_MASTER_KEY` enables gateway signature middleware; exact deployed wire-scheme compatibility remains unconfirmed (`Q-SIG`). Do not claim broader replay protection than the implemented timestamp drift check.
- Kafka password/JKS/PEM material is secret. Truststore content is parsed inside `messaging/internal/kafkatls` and never logged.
- HTTP identity propagation is the caller's responsibility: `WithInternalPropagation()` forwards identity headers but does not validate the destination host. This is an accepted trust-boundary risk (`GC-27`).
- Kafka request/user/registered label identity is restored from message headers trusted on the internal Kafka network. The absence of per-message identity authentication is an accepted risk (`GC-27`).
- Diagnostic snapshots contain bounded function/file locations and numeric runtime data; goroutine arguments are not collected.

Evidence: `config/load.go`, `server/gateway/signature_middleware.go`, `messaging/internal/kafkatls/truststore.go`, `httpclient/client.go`, `messaging/kafka_consumer.go`, `observability/diagnostics/diagnostics.go`.

## Preserve open decision
- Keep `Q-SIG` open until a consuming service confirms its current signature scheme.
