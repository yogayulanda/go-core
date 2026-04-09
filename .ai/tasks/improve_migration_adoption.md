Status: done

Task: improve migration adoption workflow

Goal:
make migration behavior easier to adopt and safer to operate as more services consume `go-core`

Scope Layers:

runtime
tests
docs
ai

Allowed Paths:

migration/
docs/
MIGRATION.md
.ai/
README.md

Constraints:

keep migration execution explicit
preserve lock safety semantics
improve runtime signals and upgrade discipline without hidden automation

Expected Output:

- clearer migration adoption guidance
- stronger tests/docs around lock and autorun behavior
- upgrade notes that match real public behavior

Implemented Notes:

- migration autorun now has additive logger-aware variants without changing explicit ownership
- logger-aware autorun emits `migration_autorun` and `migration_lock` service logs
- tests cover skipped and successful logger-backed autorun behavior
- migration guidance now documents the logger-aware adoption path and unchanged lock semantics
