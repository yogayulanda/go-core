# Observability

Logs:
- structured keys
- include `request_id`
- redact sensitive data

Metrics:
- request count + latency (HTTP/gRPC)
- optional transaction counters

Tracing:
- OTEL optional
- propagate context across layers
