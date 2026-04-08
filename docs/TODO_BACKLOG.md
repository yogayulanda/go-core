# TODO Backlog

This backlog lists improvement areas that are still untouched or only lightly touched relative to the newer foundation direction.

## P1

- messaging and outbox runtime:
  bring publisher/consumer/worker observability and ownership up to the same standard as DB/logging
- examples refresh:
  upgrade older examples so they reflect the current golden path, not older partial patterns

Recently completed:

- runtime orchestration:
  `app/` and `server/` now emit aligned `ServiceLog` for init, startup, readiness, orchestration, shutdown, and component failure signals
- transport alignment:
  gRPC and gateway now emit aligned request ID, request metrics, service metrics, and structured `ServiceLog`

## P2

- cache ergonomics and observability:
  align Redis and Memcached behavior with DB-style health and service logging expectations
- error contract hardening:
  strengthen REST/gRPC mapping consistency and service-facing error guidance
- security observability:
  make extractor/JWT behavior easier to operate and diagnose
- migration adoption workflow:
  improve migration runtime signals and upgrade guidance

## P3

- resilience observability:
  connect retry/timeout helpers with better logging and usage guidance
- script and gate alignment:
  evolve local scripts and lint gates to reflect foundation maturity
- versioning and upgrade discipline:
  define clearer version/change expectations for future multi-service adoption
- legacy `.ai` task cleanup:
  refresh older tasks that predate the current foundation direction
