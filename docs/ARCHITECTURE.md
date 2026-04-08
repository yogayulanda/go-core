# Architecture

## Shape
- Core contracts:
  config, lifecycle, error, messaging interfaces, and selected platform-standard technical contracts.
- Core adapters: DB/cache/kafka/grpc/gateway.
- Service code: domain/usecase/repository/transport.

## Runtime Path
1. load config
2. validate config
3. optional migration autorun
4. build `app.App`
5. build grpc + gateway
6. run via `server.Run(...)`
7. graceful shutdown via lifecycle

Runtime orchestration signals:
- `app.New(...)` emits `ServiceLog` for `app_init`
- `app.Start(...)` and lifecycle shutdown emit `ServiceLog` for runtime start, shutdown request, shutdown hook failure, and final shutdown status
- `server.Run(...)` emits `ServiceLog` for orchestration start, component start result, shutdown request, and final runtime result
- `server.LogStartupReadiness(...)` emits readiness `ServiceLog` for gRPC, gateway, and combined service readiness

Transport alignment:
- gRPC interceptors emit request ID, request metrics, additive service metrics, and `ServiceLog` for `grpc_request`
- HTTP gateway middleware emits request ID, HTTP metrics, additive service metrics, and `ServiceLog` for `http_request`
- transport wrappers keep compact external errors while internal observability stays structured

## Contract Classes
- Generic foundation contracts:
  reusable runtime/bootstrap building blocks for all services.
- Platform-standard technical contracts:
  intentionally shared technical standards for a class of services.

Current example:
- `logger.TransactionLog` for transaction-oriented service monitoring.

## Extension Rule
Add only when reused by multiple services.
