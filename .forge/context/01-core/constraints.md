---
id: core.constraints
title: Constraints
type: core
status: inferred
confidence: medium
source: ai
evidence:
  - { type: code, ref: config/validate.go }
  - { type: code, ref: app/app.go }
  - { type: code, ref: migration/goose.go }
  - { type: code, ref: messaging/outbox/repository.go }
  - { type: code, ref: messaging/outbox/worker.go }
  - { type: doc, ref: docs/DOMAIN.md }
owner: unresolved
updated: 2026-05-21
---

# Constraints

## Domain Boundary

Layer: business intent. `go-core` must remain domain-agnostic and must not own business entities, business rules, or service-specific states.

## Validation Semantics

Layer: config validation in `config/validate.go`.
- `SERVICE_NAME` is required.
- Each configured DB alias requires `DB_<N>_DRIVER` and a resolvable DSN/composed DSN.
- `MIGRATION_DB` and `MIGRATION_DIR` are required only when `MIGRATION_AUTO_RUN=true`; migration DB must exist in `DB_LIST`.
- Kafka brokers, Redis address, Memcached servers/timeout, JWT public key, TLS cert/key, and JWT include/exclude method rules are conditional on their feature flags.

Layer: runtime fallback/behavior.
- Optional databases that fail initialization do not block startup when `DB_<N>_REQUIRED=false`.
- `migration` default lock key falls back to `<SERVICE_NAME>:migration:<MIGRATION_DB>` when unset.
- `messaging/outbox` sets generated event ID, `PENDING` status, and `created_at` at publish time; worker sets `published_at` after successful publish.

## DB Constraints

No application-owned migration files are present, so DB-level `NOT NULL`, `CHECK`, `UNIQUE`, or FK constraints are unknown except for the internal SQL Server `dbo.goose_db_version` metadata table created in `migration/goose.go`.

## Generated Code Policy

No committed `gen/`, `generated/`, protobuf generated source, or equivalent generated-code directory was found in repo source. `.forge/context/generated/` is not source of truth.
