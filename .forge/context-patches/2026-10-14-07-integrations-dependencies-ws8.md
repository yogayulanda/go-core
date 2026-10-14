# Proposed Context Patch: 07 Integrations And Dependencies

Status: promoted on 2026-10-14 to `.forge/context/07-integrations-and-dependencies.md`.

## Replace stale confirmed facts
- Kafka supports SASL PLAIN, SCRAM-SHA-256, and SCRAM-SHA-512 (not PLAIN only).
- Kafka TLS material accepts JKS or PEM; the JKS reader is pinned to `github.com/pavlo-v-chernykh/keystore-go/v4 v4.5.0` and imported only by `messaging/internal/kafkatls`.

## Add confirmed facts
- The same authenticated Kafka dialer/transport configuration is used for producer, consumer, HTTP readiness, and gRPC health checks.
- The dependency is MIT licensed and attributed in `NOTICE.txt`; it is used by `messaging/internal/kafkatls` for JKS parsing. The current advisory inventory reports no finding against this dependency.
- Current residual module and standard-library advisories are listed in `docs/SECURITY_ADVISORIES.md`.
- The deployment's selected Kafka SASL mechanism remains to be confirmed operationally; implementation supports PLAIN, SCRAM-SHA-256, and SCRAM-SHA-512.

Evidence: `messaging/kafka_auth.go`, `messaging/internal/kafkatls/truststore.go`, `config/validate.go`, `server/gateway/health.go`, `server/grpc/server.go`, `NOTICE.txt`.
