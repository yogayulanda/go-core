# go-core Workflow

## Official Development Workflow
1. Read `.ai/context.md`, `.ai/architecture.md`, and this file first.
2. Confirm the requested change belongs in `go-core` and not in a consuming service.
3. Identify whether the change affects:
   - public API
   - config/env contract
   - runtime behavior
   - transport-facing behavior
   - observability contract
4. Make the smallest change that satisfies the task.
5. Update docs, tests, and `MIGRATION.md` when required.
6. Verify with repository checks before considering the work complete.

## Execution Rules
- prefer additive changes
- keep public surface area small
- do not redesign architecture unless explicitly required
- preserve explicit ownership of optional runtime components
- keep examples and docs aligned with the golden bootstrap path
- avoid product-specific naming and assumptions

## Task Execution Workflow
For any non-trivial task:
1. inspect relevant code and docs
2. define the exact boundary of change
3. implement only within allowed scope
4. run focused tests first, then broader verification as needed
5. confirm README/docs/examples still describe the actual behavior
6. update `.ai` context when repository expectations change

## Release Discipline
For consumer-visible changes:
- update `README.md`
- update relevant docs
- update tests
- update `MIGRATION.md` when upgrade behavior changes
- review version impact against `docs/VERSIONING.md`

Before release:
- run CI baseline
- run `make quality-gate`
- collect release evidence from `docs/RELEASE_EVIDENCE_TEMPLATE.md`
- confirm release metadata uses `version.Version`, `version.Commit`, and `version.BuildDate`

## Acceptance Standard
A task is complete only when:
- implementation is bounded
- docs are aligned
- tests pass
- migration notes are updated when needed
- `.ai` context remains accurate for future automated work
