# Change Checklist

Use this checklist before merging changes that affect `go-core` as a foundation repo.
Starting at `v1.0.0`, use it as the release-discipline checklist for semver-governed changes.

## Public Contract Change

- [ ] README updated
- [ ] relevant docs updated
- [ ] `.ai/` context and tasks updated
- [ ] tests updated
- [ ] `MIGRATION.md` updated when upgrade behavior changes
- [ ] release version impact reviewed against `docs/VERSIONING.md`

## Runtime Change

- [ ] hidden lifecycle behavior not introduced
- [ ] optional dependency behavior still explicit
- [ ] readiness impact reviewed
- [ ] logging, metrics, and tracing impact reviewed

## Adoption Change

- [ ] examples/templates still reflect the golden path
- [ ] service-facing behavior stays understandable for new adopters
- [ ] versioning or upgrade note added if needed

## Release Metadata

- [ ] release build injects `version.Version`
- [ ] release build injects `version.Commit`
- [ ] release build injects `version.BuildDate`
