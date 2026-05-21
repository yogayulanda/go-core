---
id: knowledge.inferred
title: Inferred Knowledge
type: knowledge
status: inferred
confidence: medium
source: ai
evidence:
  - { type: doc, ref: README.md }
  - { type: code, ref: go.mod }
owner: unresolved
updated: 2026-05-21
---

# Inferred Knowledge

| ID | Priority | Claim | Evidence | Owner | Created | Status |
|---|---|---|---|---|---|---|
| I-001 | important | `go-core` is a reusable Go foundation library, not a deployable service. | `README.md`, `go.mod` | unresolved | 2026-05-21 | inferred |
| I-002 | important | Active engineering layers are backend and testing only. | Go source tree, `Makefile`, `.github/workflows/ci.yml` | unresolved | 2026-05-21 | inferred |
| I-003 | important | Infrastructure layer is not active because no Helm/Terraform/Kubernetes/deploy pipeline evidence was found. | `.github/workflows/ci.yml`, repo file scan | unresolved | 2026-05-21 | inferred |
| I-004 | important | Runtime data ownership is limited: no app-owned migration files; `goose_db_version` is internal metadata and `outbox_events` is an expected external table contract. | `migration/goose.go`, `messaging/outbox/*.go` | unresolved | 2026-05-21 | inferred |
| I-005 | important | Config validation is conditional by feature flag and should not be flattened into globally required fields. | `config/validate.go` | unresolved | 2026-05-21 | inferred |
| I-006 | informational | Legacy `.ai/` artifacts are useful references but not authoritative without human confirmation. | `.ai/*`, code cross-check | unresolved | 2026-05-21 | inferred |

## Legacy Artifact Classification

| Artifact | Classification | Treatment |
|---|---|---|
| `.ai/` | useful-reference | Use only when aligned with repo code/docs; code wins on conflict. |
| `CLAUDE.md` | unknown-authority | Root adapter/context note existed before this init; keep as reference, not confirmation. |
| `docs/` | useful-reference | Strong repository evidence when aligned with code. |
| `README.md`, `MIGRATION.md`, `PROJECT_CONTEXT.md`, `CONTEXT.md`, `AI_RULES.md` | useful-reference | Evidence source, not automatic human confirmation. |
| `.cursor/`, `.claude/`, `AGENTS.md` | not-found | No action. |
