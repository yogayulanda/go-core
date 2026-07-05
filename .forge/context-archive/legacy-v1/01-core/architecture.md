---
id: core.architecture
title: Architecture
type: core
status: inferred
confidence: medium
source: ai
evidence:
  - { type: doc, ref: README.md }
  - { type: doc, ref: docs/ARCHITECTURE.md }
  - { type: code, ref: app/app.go }
  - { type: code, ref: server/run.go }
  - { type: code, ref: server/grpc/server.go }
  - { type: code, ref: server/gateway/gateway.go }
owner: unresolved
updated: 2026-05-21
---

# Architecture

## Style

Single Go module library. It provides reusable packages for service runtime composition rather than a deployable application.

## Major Components

- `config`: environment loading and validation.
- `app`: dependency container and lifecycle orchestration.
- `server/grpc` and `server/gateway`: transport wrappers, middleware, metrics, error mapping, and built-in HTTP endpoints.
- `logger`, `observability`, `errors`, `security`: shared technical contracts.
- `database`, `dbtx`, `migration`: SQL connection, transaction, and goose migration helpers.
- `cache`, `messaging`, `messaging/outbox`, `httpclient`, `resilience`: optional runtime integrations and client-side helpers.

## Runtime Path

Documented path: load config, validate config, optionally run migrations, build `app.App`, build gRPC/gateway, run via `server.Run(...)`, then graceful shutdown through lifecycle.

## APIs and Handlers

This repo does not define business RPCs or application controllers. It exposes transport wrappers and gateway endpoints registered by `server/gateway`: `/health`, `/ready`, `/version`, `/metrics`, and optional `/debug/pprof/*`.

## Data and Persistence

No service-owned migration files or business tables are present in the repo. `migration/goose.go` may create SQL Server `dbo.goose_db_version` as a goose/internal migration metadata table. `messaging/outbox` expects a consuming-service table named `outbox_events` but this repo does not include its migration.

Table roles:
- `dbo.goose_db_version`: generated/internal, created only for SQL Server migration metadata.
- `outbox_events`: external operational table expected by consumers; runtime writes/selects are implemented here, schema ownership is not evidenced in this repo.

## Integrations

Evidenced dependencies include GORM SQL Server, goose, gRPC, grpc-gateway, OpenTelemetry, Prometheus, zap, Redis, Memcached, Kafka, JWT/JWKS, resty, and circuit breaker/retry libraries.
