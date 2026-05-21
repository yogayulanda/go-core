---
id: meta.glossary
title: Glossary
type: meta
status: inferred
confidence: medium
source: ai
evidence:
  - { type: doc, ref: README.md }
  - { type: doc, ref: docs/DOMAIN.md }
  - { type: code, ref: dbtx/manager.go }
  - { type: code, ref: migration/goose.go }
owner: unresolved
updated: 2026-05-21
---

# Glossary

All entries: `status: inferred`, `source: ai`.

| Term | Canonical Definition |
|---|---|
| `go-core` | Reusable Go foundation library for service runtime/bootstrap concerns. |
| Foundation contract | Generic or platform-standard technical contract intended for reuse by services. |
| `ServiceLog` | Structured technical service-flow log contract. |
| `DBLog` | Structured database operational/query log contract. |
| `TransactionLog` | Platform-standard transaction-flow monitoring log for transaction-oriented services; not a business rule container. |
| `dbtx` | SQL transaction orchestration helper; distinct from `TransactionLog`. |
| `outbox_events` | External table contract expected by outbox helpers; schema is not owned in this repo evidence. |
| `goose_db_version` | Goose/internal migration metadata table created by SQL Server migration helper. |
