# Versioning and Upgrade Discipline

`go-core` uses a lightweight release discipline suitable for a stable reusable foundation repo.

## Versioning intent

- semantic versioning applies starting at `v1.0.0`
- `v1.0.0` is the first compatibility baseline for downstream service adoption
- tagged releases are the source of truth for adoption
- release builds should set:
  - `version.Version`
  - `version.Commit`
  - `version.BuildDate`
- `/version` should expose the same release metadata that was built into the binary

## Change classes

- patch release:
  internal fixes, test improvements, docs alignment, or non-breaking operational clarifications
- minor release:
  additive foundation capability or additive public contract that existing adopters can ignore safely
- major release:
  breaking change to exported APIs, config behavior, runtime behavior, transport-facing contracts, or other service-consumer-visible expectations

## Upgrade note discipline

Update `MIGRATION.md` when a change affects upgrade behavior for consuming services, including:

- exported API signature changes
- removed or renamed config/env keys
- changed runtime behavior that services must account for
- changed transport-facing error or auth behavior
- changed observability contract that operators or dashboards depend on

Do not add entries for internal-only cleanup that does not change consumer-visible behavior.

## Merge and release checks

Before merging a public contract change:

- update README and relevant docs
- update tests
- update `.ai/` source-of-truth context when repo expectations changed
- update `MIGRATION.md` when the change alters upgrade behavior

Before cutting a release:

- run the CI baseline
- run `make quality-gate`
- collect release evidence using `docs/RELEASE_EVIDENCE_TEMPLATE.md`
- verify `/version` exposes the injected release metadata for the build being tagged
