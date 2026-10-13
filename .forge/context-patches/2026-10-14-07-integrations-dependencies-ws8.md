# Proposed Context Patch: 07 Integrations And Dependencies

Status: proposal only; curated context remains unchanged.

## Replace stale confirmed facts
- Kafka supports SASL PLAIN, SCRAM-SHA-256, and SCRAM-SHA-512 (not PLAIN only).
- Kafka TLS material accepts JKS or PEM; the JKS reader is pinned to `github.com/pavlo-v-chernykh/keystore-go/v4 v4.5.0` and imported only by `messaging/internal/kafkatls`.

## Add confirmed facts
- The same authenticated Kafka dialer/transport configuration is used for producer, consumer, HTTP readiness, and gRPC health checks.
- The dependency is MIT licensed and attributed in `NOTICE.txt`; upstream is low-churn (last release ~3 years, latest commit ~2 years), accepted by the human for stable JKS format; `govulncheck` reported no finding against this dependency.
- `govulncheck` reported existing findings in other dependency versions; see the combined execution report/backlog.
- Office Kafka SASL mechanism remains to be confirmed operationally; implementation supports all approved choices.

Evidence: `messaging/kafka_auth.go`, `messaging/internal/kafkatls/truststore.go`, `config/validate.go`, `server/gateway/health.go`, `server/grpc/server.go`, `NOTICE.txt`.
