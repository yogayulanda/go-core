Repository Map

Top-level modules:

- `app`: build and own the runtime container
- `config`: parse env vars into typed config
- `database`: open health-checked SQL pools
- `dbtx`: transaction manager and context helpers
- `migration`: migration autorun and locking
- `server`: gRPC/gateway orchestration
- `errors`: error mapping helpers
- `logger`: structured technical logging and redaction
- `cache`: Redis and Memcached adapters
- `messaging`: Kafka abstraction and outbox support
- `observability`: tracing, metrics, request ID
- `security`: auth extraction and JWT validation
- `resilience`: timeout and retry helpers
- `templates`, `examples`: downstream implementation guidance

High-risk change areas:

- `config/`
- `app/`
- `migration/`
- `server/`

Any change there should consider compatibility and README drift.

Boundary-sensitive change areas:

- `errors/`: affects transport-facing contracts
- `security/`: affects cross-service auth behavior
- `logger/` and `observability/`: affects operational defaults across all consumers
