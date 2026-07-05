# Integrations and Dependencies

## When to Read
- Read before changing provider libraries, dependency startup expectations, or service-consumer integration guidance.

## Do Not Use This For
- Auth contract detail: `08-security-and-access.md`.
- Error mapping policy: `09-errors-and-resilience.md`.

## Source of Truth
- Third-party libraries, optional infra dependencies, and documented service-consumer patterns around them.

## Current Context
- The foundation integrates with SQL databases, Redis, Memcached, Kafka, gRPC, grpc-gateway, OpenTelemetry, Prometheus, Zap, JWT/JWKS verification, and Resty-based outbound HTTP.

## Confirmed Facts
- `go.mod` declares first-party dependency surfaces including `gorm`, `goose`, `segmentio/kafka-go`, `redis/go-redis/v9`, `gomemcache`, `grpc`, `grpc-gateway`, `prometheus/client_golang`, `otel`, `zap`, `resty`, `jwt/v5`, and `keyfunc/v3`.
- Infra dependencies are opt-in via configuration profiles: Redis, Memcached, Kafka, tracing export, JWT verification, HTTP signatures, and startup migration.
- `app.NewKafkaPublisher(...)` and `app.NewKafkaConsumer(...)` wire default logger and metrics options, but topic choice, retry/DLQ policy, concurrency, and worker ownership remain service decisions.
- The outbound HTTP client and resilience helpers are intended for bounded timeout, retry, and circuit-breaker behavior around external services.
- Kafka configuration supports SASL Plain authentication and JKS certificates according to `docs/MESSAGING_PATTERN.md` and `docs/CONFIGURATION_PROFILES.md`.

## Assumptions
- None.

## Related Files
- `04-interfaces-and-contracts.md`
- `09-errors-and-resilience.md`
- `12-runtime-deployment-and-config.md`
