# Runtime Deployment and Config

## When to Read
- Read before changing env vars, startup requirements, release metadata, or deployment-facing runtime behavior.

## Do Not Use This For
- Manual recovery workflow detail: `13-operations-and-runbook.md`.
- High-level rationale and non-goals: `14-decisions-assumptions-and-constraints.md`.

## Source of Truth
- Runtime configuration model, enablement profiles, release metadata expectations, and deployment-facing constraints exposed by the repo.

## Current Context
- The module declares the minimum Go version required by its dependency graph (`go 1.26.0`); CI and release validation use the Go 1.26.9 patch toolchain. Runtime configuration is primarily environment-driven.

## Confirmed Facts
- `SERVICE_NAME` and `APP_ENV` are required by validation. `LOG_LEVEL`, `GRPC_PORT`, and `HTTP_PORT` have defaults; `SHUTDOWN_TIMEOUT` is not an environment override. `LOG_TIMEZONE` configures logger timestamp rendering.
- Relevant environment variables include `APM_ENDPOINT`, `DIAG_ENABLED`, `METRICS_ENABLED`, `HTTP_PPROF_ENABLED`, Kafka SASL/JKS settings, and database settings. `METRICS_ENABLED` defaults false; when false, metric recording is gated and the registered `/metrics` route returns 404.
- Optional profiles cover transport TLS, SQL databases, migration autorun and locking, tracing export, Redis, Memcached, Kafka, internal JWT auth, and HTTP request-signature validation settings.
- Validation guidance distinguishes `cfg.Validate()` for simple errors from `cfg.ValidateIssues()` for structured validation feedback.
- Release builds are expected to inject `version.Version`, `version.Commit`, and `version.BuildDate`, and `/version` should expose the same values.
- CI is defined in GitHub Actions, but production rollout topology, deployment manifests, and rollback automation are not part of this repository.
- Defaults include `DrainDelay=3s`, `ShutdownTimeout=25s`, HTTP read-header/read/write/idle timeouts of 5s/15s/60s/120s, and a 1 MiB body limit. Diagnostics default to 80% heap, 5000 goroutines, a 10-minute cooldown, and a 30-second interval.
- `KAFKA_JKS_FILE` accepts JKS or PEM trust material; supported SASL mechanisms are PLAIN, SCRAM-SHA-256, and SCRAM-SHA-512.

## Assumptions
- Deployment environment, secret distribution, and runtime process topology are consumer-service responsibilities.

## Related Files
- `01-service-overview.md`
- `11-testing-and-quality.md`
- `13-operations-and-runbook.md`
