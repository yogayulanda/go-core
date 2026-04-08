Feature Map

- app container and lifecycle orchestration with structured service logs: `app/`, `server/run.go`, `server/startup.go`
- env config loading: `config/load.go`
- config validation and alias normalization: `config/validate.go`, `config/db_alias.go`, `docs/CONFIGURATION_PROFILES.md`
- DB initialization: `database/sqlserver.go`, `app/app.go`
- transaction context propagation: `dbtx/`
- migration autorun: `migration/goose.go`
- readiness and startup behavior: `server/`
- gRPC transport and interceptors with aligned request/service metrics and `ServiceLog`: `server/grpc/`
- HTTP gateway endpoints and middleware with aligned request/service metrics and `ServiceLog`: `server/gateway/`
- logger baseline and structured logging flavors: `logger/`
- transaction observability contract: `logger/transaction.go`, `observability/metrics.go`, `docs/TRANSACTION_OBSERVABILITY.md`
- tracing and metrics bootstrap: `observability/`
- auth extraction and JWT verification: `security/`
- cache adapters: `cache/`
- messaging abstraction and outbox: `messaging/`, `messaging/outbox/`
- messaging/outbox consumption pattern: `docs/MESSAGING_PATTERN.md`, `examples/outbox_example.go`
- timeout and retry helpers: `resilience/`
- examples and implementation templates: `examples/`, `templates/`, `docs/SERVICE_BOOTSTRAP.md`
