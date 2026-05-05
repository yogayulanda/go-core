# .ai/ — AI Context for go-core

This folder contains the structured AI context for all work in the `go-core` repository.

## Reading Order

Always start here, in order:

1. **`.ai/context.md`** — project overview, key modules, critical rules
2. **`.ai/architecture.md`** — system design, layer model, scaling considerations
3. Continue to the specific file for the area being worked on

## Navigation by Task Area

| Working on... | Read |
|---|---|
| Auth / JWT / security | `.ai/security.md` |
| DB transactions / idempotency / outbox | `.ai/transactions.md` |
| Module APIs / symbols / constructors | `.ai/modules.md` |
| Request flow / observability / logging | `.ai/data-flow.md` |
| External integrations / env config | `.ai/integrations.md` |
| Code style / naming / patterns | `.ai/conventions.md` |
| Why design decisions were made | `.ai/decisions.md` |
| Dev workflow / acceptance checklist | `.ai/workflow.md` |

## Folder Structure

```
.ai/
├── context.md        ← READ THIS FIRST
├── architecture.md
├── security.md
├── transactions.md
├── modules.md
├── data-flow.md
├── integrations.md
├── conventions.md
├── decisions.md
├── workflow.md
│
├── AI_RULES.md       ← Compact rules & review checklist
├── AI_WORKFLOW.md    ← Ownership principles & prompt roles
├── STATUS.md         ← Task progress tracking
├── config.yaml       ← AI context file index
│
├── prompts/          ← Ready-to-use prompt templates (8 files)
└── tasks/            ← Task definitions for bounded changes
```

## Maintenance Rule

Update the relevant `.ai/` file whenever public behavior changes.
`.ai/` is the single source of truth for AI context in this repo.
