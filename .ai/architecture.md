# go-core Architecture

## Architectural Shape
`go-core` is a framework/library repository, not a business service.

Core layers:
1. App container and lifecycle
2. Config loading and validation
3. Transport wrappers
4. Observability and logging baseline
5. Infra connectors and operational helpers
6. Shared technical contracts
7. Examples and templates for downstream adoption

## Stable Runtime Path
1. load config
2. validate config
3. optionally run startup migration
4. build `app.App` with `app.New(ctx, cfg)`
5. build and register gRPC and/or HTTP gateway
6. run via `server.Run(...)`
7. shut down through lifecycle ownership

## Stable Runtime Contracts
Runtime orchestration:
- `app.New(...)` emits `app_init`
- `app.Start(...)` emits runtime start and shutdown-request signals
- lifecycle emits `lifecycle_shutdown` and `shutdown_hook`
- `server.Run(...)` emits runtime orchestration and component-start signals
- `server.LogStartupReadiness(...)` emits readiness signals for startup visibility

Transport boundary:
- gRPC wrappers own request ID propagation, auth extraction/verification, request metrics, service metrics, and `grpc_request` service logs
- HTTP gateway wrappers own request ID propagation, HTTP panic recovery, HTTP payload signature validation, HTTP metrics, service metrics, and `http_request` service logs
- transport wrappers keep external errors compact and internal diagnostics structured

Observability baseline:
- `logger.ServiceLog` is the default technical service-flow log
- `logger.DBLog` is the default DB operational/query log
- `logger.TransactionLog` is only for transaction-oriented services
- metrics and tracing remain additive and framework-level

## Boundary Rules
Keep in `go-core`:
- reusable runtime/bootstrap behavior
- explicit infra integration contracts
- technical observability and error contracts

Keep out of `go-core`:
- business policies
- service-specific domain workflows
- product-specific defaults, aliases, or event semantics
- hidden background behavior not controlled by the consuming service

## Change Sensitivity
High-sensitivity areas:
- `app/`
- `config/`
- `server/`
- `migration/`
- `errors/`
- `security/`
- `logger/`
- `observability/`

Any change in these areas must be reviewed as a public-contract risk first.
