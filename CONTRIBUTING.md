# Contributing

Thanks for helping improve `go-core`. This repository is a production-minded Go foundation, so changes should favor clarity, operational readability, and bounded abstractions.

## Development Expectations

- Keep Go idiomatic and direct.
- Preserve existing package boundaries unless there is a clear maintenance or correctness reason to change them.
- Prefer explicit behavior over hidden magic.
- Keep runtime behavior observable through logs, metrics, traces, errors, or tests where appropriate.
- Update docs when public behavior, configuration, migration behavior, or operational expectations change.
- Avoid product-specific business logic in this repository.

## Before Opening a PR

Run the fast local checks:

```bash
make test
make vet
make lint
```

For broader changes, run:

```bash
make quality-gate
```

Add or update tests when changing behavior. Narrow documentation-only changes do not require new tests, but examples and commands should remain accurate.

## Review Guidelines

Good changes usually:

- have a small, understandable scope
- name concepts clearly
- keep service/domain/repository/transport responsibilities separated
- explain operational impact in the PR description
- include migration notes when public upgrade behavior changes

Please avoid:

- framework creep
- speculative architecture rewrites
- unnecessary abstraction layers
- giant PRs without operational justification
- fake demo flows that do not reflect actual package behavior
- adding secrets, private endpoints, or customer/production data to tests, docs, fixtures, or examples

## Compatibility

Public contract changes should be intentional and documented. Update `MIGRATION.md`, `CHANGELOG.md`, and relevant docs when a change affects consumers.
