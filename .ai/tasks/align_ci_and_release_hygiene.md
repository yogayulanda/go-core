Status: done

Task: align ci and release hygiene

Goal:
make repository quality gates visible both in CI and in release docs

Scope Layers:

ci
docs

Allowed Paths:

.github/
docs/
README.md
Makefile
scripts/

Constraints:

CI should run `go test ./...`, `go vet ./...`, and `golangci-lint run`
keep stronger local quality gate with race testing and gosec
avoid changing runtime behavior

Expected Output:

- baseline CI workflow
- release docs aligned with local and CI gate expectations
