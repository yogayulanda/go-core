# TODO Backlog

- JWKS fallback to a static key when the endpoint is unavailable requires a separate design (fallback transition, return to JWKS, and logging).

This backlog tracks work intentionally left outside the completed foundation updates. Do not interpret deferred items as implemented defaults.

## Deferred Decisions

- Downstream forks may choose an import module path before adoption; this does not block the public release.
- Q-SIG confirmation is deferred and is not a release blocker; do not change the current signature contract without compatibility evidence.

## Operational Follow-Up

- Test tidak idempoten saat `-count>1`: 15 tests failed on repeated runs because tests share process-global state. Suspected causes by area: response-code registry tests do not isolate or unregister global registrations; gateway and messaging metrics tests reuse global Prometheus collectors/registries; Kafka observability tests retain global metric state; outbox worker metric assertions depend on prior global observations; HTTP client identity/deadline behavior and concurrent error lookup may share mutable package state or timing-sensitive state. Reproduce each test in isolation and repeated, then add reset/isolation fixtures; do not rely on this list as a confirmed root-cause diagnosis.
	- `TestClient_PropagatesIdentityUsesDeadlineAndDoesNotRetry`
	- `TestCustomErrorHandler_RegisteredHTTPStatusOverridesGRPCDefault`
	- `TestHTTPMetricsUseMatchedTemplateAndBoundUnmatchedRoutes`
	- `TestKafkaConsumer_ObservabilityCommitFailed`
	- `TestKafkaConsumer_ObservabilityDLQSuccessAndFetchFailed`
	- `TestKafkaConsumer_ObservabilitySuccess`
	- `TestKafkaPublisher_ObservabilityFailureAndDLQ`
	- `TestKafkaPublisher_ObservabilitySuccess`
	- `TestLookupConcurrentReads`
	- `TestMessagingMetricRecordingDisabledByDefault`
	- `TestWorkerRunOnce_CommitFailure_ReturnsError`
	- `TestWorkerRunOnce_EmptyBatch_EmitsMetric`
	- `TestWorkerRunOnce_PublishFailure_ReturnsError`
	- `TestWorkerRunOnce_SuccessBatch_EmitsMetric`
	- `TestSuccessEnvelope_PendingCodeUsesRegistry`
- Run `go test -tags=integration ./messaging/outbox/...` against disposable SQL Server and MySQL instances. Integration tests compile but were not run in the implementation environment because Docker was unavailable (S8).
- Confirm which SASL mechanism the deployment's Kafka cluster uses. The library supports PLAIN, SCRAM-SHA-256, and SCRAM-SHA-512.
- Reassess residual dependency advisories in `SECURITY_ADVISORIES.md` on each Go toolchain patch and minor upgrade.
- Run the interleaved hot-path benchmark suite on a stable Linux CI runner and retain benchstat evidence as a regression guard; WSL is too noisy for reliable sub-10% decisions.
- Add an end-to-end gateway benchmark that traverses grpc-gateway to a gRPC handler over bufconn; the current `httptest.NewRecorder` benchmark stops at an in-process HTTP handler.
- Successful plain `runtime.ServeMux.HandlePath` handlers that do not call gRPC currently have no access log. gRPC-backed routes log in the gRPC interceptor and Raw routes log in the envelope boundary. Decide whether to add a single gateway access log for non-Raw direct handlers in a follow-up; leave behavior unchanged for v1.2.0.
- Confirm deployment termination grace period exceeds `DrainDelay + ShutdownTimeout` (default total 28 seconds, below the common 30-second value but leaves little margin).

## Deferred Platform Work

- Rolling N-1 Go policy: use the latest patch of Go N-1, advance minor when Go releases in February/August, and apply security patches within 14 days. Set the `go` directive to the minimum required by the selected dependency graph (`go mod tidy`), not the installed patch version.
- GitLab CI (`GC-02`): add the separately approved GitLab pipeline; current CI remains GitHub Actions.
- Keep the pinned golangci-lint release compatible with the current Go toolchain and config schema.
- Inbox helper (`GC-37`) and protovalidate integration (`GC-38`) remain deferred.
- `GC-23` and `GC-24` are resolved: OTLP to Elastic APM Server without an agent; metrics default false, endpoint/recording gated, transaction counter removed.
- Dashboard, alert thresholds, data retention, and client-IP privacy retention remain deployment/platform responsibilities.
