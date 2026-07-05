# Runtime Deployment and Config

## When to Read
- Read before changing env vars, startup requirements, release metadata, or deployment-facing runtime behavior.

## Do Not Use This For
- Manual recovery workflow detail: `13-operations-and-runbook.md`.
- High-level rationale and non-goals: `14-decisions-assumptions-and-constraints.md`.

## Source of Truth
- Runtime configuration model, enablement profiles, release metadata expectations, and deployment-facing constraints exposed by the repo.

## Current Context
- The module targets Go `1.24.3` and is configured primarily through environment variables.

## Confirmed Facts
- `SERVICE_NAME` is strictly required by validation, while `APP_ENV`, `LOG_LEVEL`, `SHUTDOWN_TIMEOUT`, `GRPC_PORT`, and `HTTP_PORT` are defaulted by `config.Load(...)`.
- Supported runtime config also includes `LOG_TIMEZONE` for logger timestamp rendering.
- Optional profiles cover transport TLS, SQL databases, migration autorun and locking, tracing export, Redis, Memcached, Kafka, internal JWT auth, and HTTP request-signature validation settings.
- Validation guidance distinguishes `cfg.Validate()` for simple errors from `cfg.ValidateIssues()` for structured validation feedback.
- Release builds are expected to inject `version.Version`, `version.Commit`, and `version.BuildDate`, and `/version` should expose the same values.
- CI is defined in GitHub Actions, but production rollout topology, deployment manifests, and rollback automation are not part of this repository.

## Assumptions
- Deployment environment, secret distribution, and runtime process topology are consumer-service responsibilities.

## Related Files
- `01-service-overview.md`
- `11-testing-and-quality.md`
- `13-operations-and-runbook.md`
