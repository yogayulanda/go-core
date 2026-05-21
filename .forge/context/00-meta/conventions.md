---
id: meta.conventions
title: Conventions and AI Operational Contract
type: meta
status: inferred
confidence: medium
source: ai
evidence:
  - { type: doc, ref: .forge/forge.config.yaml }
  - { type: doc, ref: CLAUDE.md }
owner: unresolved
updated: 2026-05-21
---

# Conventions

## Dominant Language

Context narrative language is English because repository docs and code comments are primarily English. User-facing summaries may be Bahasa Indonesia when requested. Technical identifiers are never translated.

## Evidence Rules

- Repository code wins over legacy AI artifacts and docs on conflict.
- `confirmed` and `inferred` content must cite concrete evidence.
- No phantom ADR references: cite only ADR files that exist.
- `.forge/context/generated/` and `.forge/context/temp/` are not source of truth.

## Layer Activation

Activate only layers with repository evidence. For this repo: `backend` and `testing` are active. `frontend`, `mobile`, and `infrastructure` are inactive; CI/test scripts alone are not deployment/IaC ownership evidence.

## Mode File Schema

Every `modes/*.md` file must expose exactly these Markdown sections after the title, in this order:

- `## include`
- `## on_demand`
- `## exclude`
- `## token_budget`
- `## notes`

`## token_budget` must contain only a decimal integer. Labels such as `medium` or `medium-high` are invalid.

Modes are context loading deltas only. They must not re-list `00-meta/*` or `01-core/*` unless explicitly needed, contain domain knowledge, contain workflow prose, or duplicate this conventions file.

## Validation Attribution

Keep service/config validation, DB constraints, repository fallback behavior, and business intent separate. Do not flatten conditional validation into global required fields.

## Ownership

Use `owner: unresolved` until the repository owner/team is explicitly confirmed. Track this once as `U-OWN`.
