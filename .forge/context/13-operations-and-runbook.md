# Operations and Runbook

## When to Read
- Read during release validation, readiness troubleshooting, or dependency failure drills.

## Do Not Use This For
- Error taxonomy authority: `09-errors-and-resilience.md`.
- Env var catalog detail: `12-runtime-deployment-and-config.md`.

## Source of Truth
- Repository-provided operational checks and release-time validation procedures.

## Current Context
- The repo provides release gates and runtime probe expectations, but it does not ship environment-specific escalation playbooks.

## Confirmed Facts
- Smoke-gate verification expects `/health`, `/ready`, and `/version` to return `200`, and `/version` must match injected build metadata.
- Failure-drill validation expects `/ready` to flip to `503` after a required dependency is stopped and return to `200` after recovery.
- Performance gate scripts support steady, spike, and soak scenarios driven by `scripts/load_gate.sh` and k6.
- Release evidence should be collected with `docs/RELEASE_EVIDENCE_TEMPLATE.md` and include command outputs, dashboards/screenshots, failure-drill timestamps, release metadata, and rollback-readiness confirmation.
- `server.LogStartupReadiness(...)` is the documented hook for emitting startup readiness logs when services want them.

## Assumptions
- Human incident escalation paths and concrete dependency stop/start commands are environment-specific and outside this repository.

## Related Files
- `10-observability-and-support.md`
- `11-testing-and-quality.md`
- `12-runtime-deployment-and-config.md`
