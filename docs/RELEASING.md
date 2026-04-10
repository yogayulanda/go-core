# Releasing

This document defines the repeatable release process for `go-core`.

## Release outputs

Use each output for a different purpose:

- GitHub Release:
  announcement-style release notes for a specific tag
- `CHANGELOG.md`:
  repository version history
- `MIGRATION.md`:
  consumer-visible upgrade notes
- `docs/RELEASE_EVIDENCE.md`:
  filled evidence for a specific release run

## Standard release flow

1. Confirm the target version and update any versioned release notes if needed.
2. Ensure `README.md`, relevant docs, `.ai/`, and tests are aligned.
3. Update `CHANGELOG.md` for the release version.
4. Update `MIGRATION.md` if upgrade behavior changed.
5. Run:
   - `make quality-gate`
6. Run staging gates as required by `docs/PRODUCTION_SIGNOFF.md`.
7. Copy `docs/RELEASE_EVIDENCE_TEMPLATE.md` to `docs/RELEASE_EVIDENCE.md` and fill it.
8. Create the release commit if needed.
9. Create and push the tag.
10. Publish a GitHub Release using the tag.

## Patch release flow

For patch releases such as `v1.0.1`:

- keep changes minimal and additive
- update `CHANGELOG.md`
- update `MIGRATION.md` only when service adopters must change something
- rerun the same gates before tagging

## Tagging reminder

Builds intended for release should inject:

- `version.Version`
- `version.Commit`
- `version.BuildDate`

The `/version` endpoint should expose the same values.
