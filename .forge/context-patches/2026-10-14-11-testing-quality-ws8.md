# Proposed Context Patch: 11 Testing And Quality

Status: promoted on 2026-10-14 to `.forge/context/11-testing-and-quality.md`.

## Add confirmed facts
- `coretest` provides a real-logger recording wrapper, fake publisher, response-registry cleanup helper, and reusable refusing/silent/close-after-headers/status HTTP fixtures.
- WS-5 transport tests and WS-3 logger GTID tests now use `coretest`; `examples/coretest_example_test.go` asserts `labels.gtid`.
- SQL Server/MySQL outbox integration tests are build-tagged `integration`; they compile but cannot be claimed passed because Docker was unavailable (S8).
- Diagnostics include periodic-check and 5000-goroutine snapshot benchmarks. The snapshot benchmark fails if its measured GC-pause upper bound exceeds 10 ms; the runtime does not use that threshold as a guard.
- Current `govulncheck` findings are inventoried in `docs/SECURITY_ADVISORIES.md`; no finding targets the JKS reader dependency.

Evidence: `coretest/`, `httpclient/client_test.go`, `logger/recording_logger_external_test.go`, `examples/coretest_example_test.go`, `messaging/outbox/integration_test.go`, benchmark results.
