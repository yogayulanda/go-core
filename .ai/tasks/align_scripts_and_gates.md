Status: pending

Task: align scripts and quality gates

Goal:
make local scripts and repo gates reflect the current maturity of `go-core` as a foundation repo

Scope Layers:

ci
docs
scripts
ai

Allowed Paths:

scripts/
Makefile
.golangci.yml
.github/
docs/
.ai/
README.md

Constraints:

avoid runtime behavior changes
keep gates actionable and fast enough for regular use
align local gates, CI, and release docs

Expected Output:

- more coherent local and CI quality workflow
- docs and scripts that describe the same expectations
