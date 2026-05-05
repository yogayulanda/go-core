# go-core · AI Workflow

## Entry Point

Before any task:
1. Read `.ai/context.md` — project overview and critical rules
2. Read `.ai/architecture.md` — system design and layer model
3. Identify which module(s) are affected → read `.ai/modules.md`
4. For auth/security changes → read `.ai/security.md`
5. For transaction/idempotency changes → read `.ai/transactions.md`
6. For integration changes → read `.ai/integrations.md`

---

## Development Workflow

### Step 1: Scope the Change

Confirm the change belongs in `go-core` and not in a consuming service.

Ask:
- Is this generic infrastructure? Or service-specific logic?
- Does this introduce product-specific naming or assumptions?
- Can the consuming service own this instead?

If service-specific → reject and keep it in the consuming service.

### Step 2: Identify Contract Risk

Determine whether the change affects:

| Affected Area | Risk Level | Action Required |
|---|---|---|
| Public API surface | 🔴 HIGH | Semver review, MIGRATION.md, README update |
| Config env vars | 🔴 HIGH | Validate backward compatibility, docs update |
| Runtime behavior | 🔴 HIGH | Test coverage, migration notes |
| Metric names/labels | 🔴 HIGH | Breaking change — coordinate with dashboards |
| gRPC interceptors | 🟡 MEDIUM | Review all consumers |
| Log field names | 🟡 MEDIUM | May break log parsers |
| Internal refactor | 🟢 LOW | Tests + no API change |

### Step 3: Implement

- Make the **smallest change** that satisfies the task
- No behavior outside the task's `allowed_paths`
- Follow conventions in `.ai/conventions.md`
- Keep `go-core` domain-agnostic

### Step 4: Verify

```bash
go test ./...          # all tests must pass
make quality-gate      # lint + vet + format check
```

### Step 5: Align Documentation

For public-contract changes:
- `README.md` — user-facing behavior description
- `docs/` — relevant framework guidance doc
- `MIGRATION.md` — upgrade instructions for consuming services
- `.ai/` — update context if module behavior or API changes

---

## Acceptance Standard

A task is **complete** only when ALL of the following are true:

- [ ] Implementation is bounded to allowed scope
- [ ] `go test ./...` passes
- [ ] `make quality-gate` passes
- [ ] Public API docs (`README.md`) are aligned
- [ ] `MIGRATION.md` updated if upgrade behavior changes
- [ ] `.ai/` context updated if module behavior changes

---

## Release Discipline

Before releasing a version:

1. Run CI baseline + `make quality-gate`
2. Collect release evidence from `docs/RELEASE_EVIDENCE_TEMPLATE.md`
3. Confirm `version.Version`, `version.Commit`, `version.BuildDate` are set via `ldflags`
4. Update `CHANGELOG.md`
5. Tag with `vX.Y.Z` per semver rules in `.ai/conventions.md`

---

## AI Execution Principles

- Prefer additive changes
- Allow bounded refactors that improve framework shape
- Avoid hidden side effects
- Preserve documented exported behavior as semver contract
- Keep defaults generic — no service-specific names
- Keep service-specific logic out of framework code
- Allow explicit platform-standard observability contracts (e.g., `TransactionLog`) when intentionally standardized
- Always pass the 5-gate review: compatibility · coupling · concurrency · scale · overengineering

---

## Prompt Roles

| Prompt | Role |
|---|---|
| `.ai/prompts/breakdown.md` | Task planner |
| `.ai/prompts/execute.md` | Framework engineer |
| `.ai/prompts/fix.md` | Debugger |
| `.ai/prompts/test.md` | Tester |
| `.ai/prompts/review.md` | Reviewer |
