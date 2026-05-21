---
id: system.go-core
title: System - go-core
type: system
system_type: library
status: inferred
confidence: medium
source: ai
evidence:
  - { type: code, ref: go.mod }
  - { type: doc, ref: README.md }
  - { type: code, ref: app/app.go }
  - { type: code, ref: config/validate.go }
  - { type: code, ref: server/grpc/server.go }
  - { type: code, ref: server/gateway/gateway.go }
  - { type: code, ref: messaging/outbox/repository.go }
  - { type: code, ref: messaging/outbox/worker.go }
owner: unresolved
updated: 2026-05-21
---

# System - go-core

## Responsibility

Reusable Go library for bootstrap/runtime foundation used by consuming services.

## Public Interfaces

- Go module packages under root directories such as `app`, `config`, `server`, `logger`, `errors`, `security`, `database`, `dbtx`, `migration`, `cache`, `messaging`, `resilience`, `httpclient`, and `observability`.
- HTTP gateway wrapper registers built-in endpoints `/health`, `/ready`, `/version`, `/metrics`, and optional `/debug/pprof/*`.
- gRPC wrapper exposes registration hook for consuming-service handlers.

## Runtime Context

`go-core` is imported and run inside consuming services. It does not contain a `main` package or service-specific deployment manifest.

## Data Semantics

- No application-owned migrations or seed data are present.
- `dbo.goose_db_version` is an internal/goose metadata table created only for SQL Server migration support.
- `outbox_events` is an external table contract expected by outbox helpers; runtime behavior inserts pending events, selects unpublished events in batches, publishes them, and marks `published_at`.

## Dependencies

Canonical dependency and integration inventory lives in `core.architecture`. This system file only records `go-core`-specific runtime boundaries and public interfaces.

## Layers

Touches `layer.backend` and `layer.testing`.
