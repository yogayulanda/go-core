---
id: core.principles
title: Principles
type: core
status: inferred
confidence: medium
source: ai
evidence:
  - { type: doc, ref: docs/FOUNDATION_BOUNDARY.md }
  - { type: doc, ref: docs/CHANGE_CHECKLIST.md }
  - { type: doc, ref: docs/VERSIONING.md }
  - { type: code, ref: app/app.go }
owner: unresolved
updated: 2026-05-21
---

# Principles

- Keep business behavior in consuming service repositories.
- Add foundation contracts only when reusable across services or intentionally platform-standard.
- Keep messaging and outbox startup explicit; `app.New(...)` does not auto-start Kafka, consumers, or outbox workers.
- Public contract changes should update README, relevant docs, tests, and `MIGRATION.md` when upgrade behavior changes.
- Runtime observability should use structured `ServiceLog`, `DBLog`, metrics, and tracing rather than ad hoc signals.

## Release Discipline

Repository docs define `v1.0.0` as the first semver-governed compatibility baseline and list quality gates in README, `docs/PRODUCTION_SIGNOFF.md`, and `docs/VERSIONING.md`.
