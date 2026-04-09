# Observability

Logs:
- structured keys
- include `request_id`
- redact sensitive data
- use `logger.ServiceLog` for structured normal service flow
- use `logger.DBLog` for structured DB operational/query logging
- use `logger.TransactionLog` only for transaction-oriented services that follow the shared transaction monitoring contract
- runtime orchestration emits `ServiceLog` for `app_init`, `app_runtime`, `lifecycle_shutdown`, `shutdown_hook`, and `runtime_orchestration`
- startup readiness emits `ServiceLog` for `grpc_readiness`, `http_gateway_readiness`, and `service_readiness`
- gRPC transport emits `ServiceLog` for `grpc_request`
- HTTP gateway emits `ServiceLog` for `http_request`
- messaging publisher emits `ServiceLog` for `message_publish`
- messaging consumer emits `ServiceLog` for `message_consume`
- outbox runtime emits `ServiceLog` for `outbox_worker` and `outbox_batch`

Metrics:
- request count + latency (HTTP/gRPC)
- service operation count + latency via:
  `app_service_operation_total{service,operation,status}`
  `app_service_operation_duration_seconds{service,operation}`
- DB operation count + latency via:
  `app_db_operation_total{service,db_name,operation,status}`
  `app_db_operation_duration_seconds{service,db_name,operation}`
- messaging publish count via:
  `app_message_publish_total{service,topic,status}`
- messaging consume count + processing duration via:
  `app_message_consume_total{service,topic,group,status}`
  `app_message_process_duration_seconds{service,topic,group}`
- outbox batch count + duration + size via:
  `app_outbox_batch_total{service,status}`
  `app_outbox_batch_duration_seconds{service}`
  `app_outbox_batch_size{service}`
- optional transaction counters via `app_transaction_total{service,operation,status}` for transaction-oriented services

Current transport/runtime mapping:
- gRPC increments request metrics and additive service metrics for `grpc_request`
- HTTP gateway increments HTTP metrics and additive service metrics for `http_request`
- DB initialization paths use `DBLog`; DB metrics remain available for callers that want stable DB instrumentation
- messaging publisher/consumer now use additive messaging counters and `ServiceLog`
- outbox worker now records explicit batch outcomes instead of silent empty/failure paths

Tracing:
- OTEL optional
- propagate context across layers

See also:
- `docs/TRANSACTION_OBSERVABILITY.md`
- `docs/MESSAGING_PATTERN.md`
