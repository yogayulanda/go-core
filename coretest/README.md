# coretest

`coretest` provides reusable test helpers for `go-core` consumers. It is intended for test code, not production runtime paths.

- `NewRecordingLogger` wraps the real `logger.Logger` encoder and exposes resolved, redacted ECS entries through `Find` and `Fields`.
- `NewFakePublisher` records messages and headers and can return a configured error.
- `WithIsolatedRegistry` snapshots response-code definitions and restores them with test cleanup. Do not use registry isolation in parallel tests because the registry is process-global.
- HTTP fixtures: `RefusingAddr`, `SilentServer`, `CloseAfterHeadersServer`, and `StatusServer`.
