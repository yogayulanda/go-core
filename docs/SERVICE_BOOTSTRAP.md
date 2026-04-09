# Service Bootstrap

Use this as the canonical onboarding path for a new service consuming `go-core`.

## Golden Path

1. Load configuration with `config.Load(...)`.
2. Validate configuration with `cfg.Validate()`.
3. Optionally run `migration.AutoRunUp(cfg)` if the service chooses startup migration.
4. Build the runtime container with `app.New(ctx, cfg)`.
5. Build and register gRPC and/or HTTP gateway transport.
6. Start everything with `server.Run(ctx, application, ...)`.

## Default Guidance

- Use `errors.AppError` as the canonical service error contract.
- Use `dbtx.WithTx(...)` at the use-case boundary when SQL transaction orchestration is needed.
- Use `logger.ServiceLog` for standard structured service-flow logging.
- Use `logger.DBLog` when DB interaction needs structured operational/query logging.
- Keep optional infra explicit:
  Redis, Memcached, Kafka, migration, and transaction logging are opt-in by the consuming service.
- If Kafka is enabled, prefer `application.NewKafkaPublisher(...)` and `application.NewKafkaConsumer(...)` so logger and metrics defaults are wired automatically.
- If using outbox, keep worker startup explicit through `StartChecked(ctx)` and use `RunOnce(ctx)` for deterministic tests or service-controlled batch execution.
- Use `server.LogStartupReadiness(...)` if startup readiness logs are needed.
- Treat `server.Run(...)` as the owner of runtime orchestration; it now emits structured lifecycle/service logs for startup, shutdown, and component failures.
- Treat gRPC and gateway wrappers as the default transport observability boundary; they emit aligned request ID, request metrics, service metrics, and `ServiceLog`.
- Prefer `examples/bootstrap_example.go` as the starter wiring reference.

## Service Layer Shape

- handler layer:
  request validation and transport mapping
- service/use-case layer:
  orchestration and transaction boundaries
- repository layer:
  storage implementation and `dbtx.FromContext(ctx)` reuse

## Transaction-Oriented Services

Only services that belong to the transaction-oriented class should use:

- `logger.TransactionLog`
- `app_transaction_total`

Other services should stay on the generic logging and request metrics baseline unless a separate platform-standard contract is defined.

## Runtime and Transport Behavior

- `app.New(...)` emits `app_init`.
- `app.Start(...)` emits `app_runtime` start and shutdown-request signals.
- lifecycle shutdown emits `lifecycle_shutdown` and `shutdown_hook` results.
- `server.Run(...)` emits `runtime_orchestration` and `component_start` results.
- gRPC transport emits `grpc_request` service logs plus request/service metrics.
- HTTP gateway emits `http_request` service logs plus HTTP/service metrics.
- messaging publisher emits `message_publish` service logs plus publish metrics.
- messaging consumer emits `message_consume` service logs plus consume/process metrics.
- outbox runtime emits `outbox_worker` and `outbox_batch` service logs plus batch metrics.

## Related Starter Assets

- `examples/bootstrap_example.go`
- `examples/service_example.go`
- `examples/repository_example.go`
- `examples/grpc_handler_example.go`
- `examples/outbox_example.go`
