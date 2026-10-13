# Proposed Context Patch: 11 Testing And Quality

Status: proposal only; curated context remains unchanged.

## Add confirmed facts
- `coretest` provides a real-logger recording wrapper, fake publisher, response-registry cleanup helper, and reusable refusing/silent/close-after-headers/status HTTP fixtures.
- WS-5 transport tests and WS-3 logger GTID tests now use `coretest`; `examples/coretest_example_test.go` asserts `labels.gtid`.
- SQL Server/MySQL outbox integration tests are build-tagged `integration`; they compile but cannot be claimed passed because Docker was unavailable (S8).
- Diagnostics include periodic-check and 5000-goroutine snapshot benchmarks. Stop if measured GC pause exceeds 10ms.
- WS-6 `govulncheck` completed with auto-downloaded Go 1.26 toolchain; it reported existing advisories in dependencies including gRPC, x/net, x/crypto, OTel, and x/text, not the new JKS reader.

Evidence: `coretest/`, `httpclient/client_test.go`, `logger/recording_logger_external_test.go`, `examples/coretest_example_test.go`, `messaging/outbox/integration_test.go`, benchmark results.
