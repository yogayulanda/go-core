Feature Map

- app container and lifecycle orchestration: `app/`, `server/run.go`, `server/startup.go`
- env config loading: `config/load.go`
- config validation and alias normalization: `config/validate.go`, `config/db_alias.go`
- DB initialization: `database/sqlserver.go`, `app/app.go`
- transaction context propagation: `dbtx/`
- migration autorun: `migration/goose.go`
- readiness and startup behavior: `server/`
- gRPC transport and interceptors: `server/grpc/`
- HTTP gateway endpoints and middleware: `server/gateway/`
- logger baseline: `logger/`
- tracing and metrics bootstrap: `observability/`
- auth extraction and JWT verification: `security/`
- cache adapters: `cache/`
- messaging abstraction and outbox: `messaging/`, `messaging/outbox/`
- timeout and retry helpers: `resilience/`
- examples and implementation templates: `examples/`, `templates/`
