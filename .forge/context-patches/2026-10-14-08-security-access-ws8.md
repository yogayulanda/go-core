# Proposed Context Patch: 08 Security And Access

Status: proposal only; curated context remains unchanged.

## Add confirmed facts
- A non-empty `AUTH_SIGNATURE_MASTER_KEY` enables gateway signature middleware; exact deployed wire-scheme compatibility remains unconfirmed (`Q-SIG`). Do not claim broader replay protection than the implemented timestamp drift check.
- Kafka password/JKS/PEM material is secret. Truststore content is parsed inside `messaging/internal/kafkatls` and never logged.
- HTTP request identity is forwarded only for explicitly internal HTTP targets. Kafka request/user/registered label headers are restored from trusted internal traffic.
- Diagnostic snapshots contain bounded function/file locations and numeric runtime data; goroutine arguments are not collected.

Evidence: `config/load.go`, `server/gateway/signature_middleware.go`, `messaging/internal/kafkatls/truststore.go`, `httpclient/client.go`, `messaging/kafka_consumer.go`, `observability/diagnostics/diagnostics.go`.

## Preserve open decision
- Keep `Q-SIG` open until a consuming service confirms its current signature scheme.
