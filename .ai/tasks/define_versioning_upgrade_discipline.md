Status: done

Task: define versioning and upgrade discipline

Goal:
prepare `go-core` for broader multi-service adoption with clearer versioning and upgrade expectations

Scope Layers:

docs
ai
versioning

Allowed Paths:

docs/
MIGRATION.md
README.md
version/
.ai/

Constraints:

avoid inventing heavy release process machinery
focus on simple, explicit upgrade discipline
align migration notes with public contract changes

Expected Output:

- clearer versioning and upgrade guidance
- migration note discipline tied to public contract changes
- `.ai` context updated to enforce the same expectations
- release metadata expectations documented through `version.Version`, `version.Commit`, and `version.BuildDate`
