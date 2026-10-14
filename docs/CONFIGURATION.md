# Configuration Reference

`config.Load` reads environment variables and returns `*config.Config`. Use environment variables for infrastructure addresses, credentials, and explicit kill switches. Set runtime behavior such as timeouts, TLS files, pool sizes, authentication policy, migration policy, and tracing options by changing Config fields in code after `Load` and before `Validate`/`app.New`.

Source of truth: `config/config.go` (Config types) and `config/load.go` (`Load` defaults and environment reads). Database aliases are normalized to lowercase in `config/db_alias.go`.

## Environment Variables

There are 21 fixed environment-variable names and seven alias-templated DB names. `SERVICE_NAME` and `APP_ENV` are required by `Config.Validate`; an empty variable uses the default shown. Invalid integer/boolean strings also use the default. Comma-separated values are trimmed around each item.

| Area | Name | Default | Required? | Purpose | Example |
|---|---|---|---|---|---|
| App | `SERVICE_NAME` | empty | Yes | Service identity, logger field, and DB default lock key. | `SERVICE_NAME=catalog-api` |
| App | `APP_ENV` | empty | Yes | One of `local`, `dev`, `sit`, `uat`, `prod`; `local` selects console logs. | `APP_ENV=local` |
| App | `LOG_LEVEL` | `info` | No | Logger threshold. | `LOG_LEVEL=debug` |
| gRPC | `GRPC_PORT` | `50051` | No | gRPC listen port. | `GRPC_PORT=50051` |
| HTTP | `HTTP_PORT` | `8080` | No | Gateway listen port. | `HTTP_PORT=8080` |
| HTTP | `HTTP_CORS_ALLOWED_ORIGINS` | empty | No | Comma-separated exact origins; empty disables CORS. | `HTTP_CORS_ALLOWED_ORIGINS=https://app.example` |
| HTTP/security | `HTTP_PPROF_ENABLED` | `false` | No | Enables pprof outside production; prod/production startup is rejected. | `HTTP_PPROF_ENABLED=true` |
| Tracing | `APM_ENDPOINT` | empty | No | Enables OTLP gRPC export when nonempty. | `APM_ENDPOINT=collector.internal:4317` |
| Metrics | `METRICS_ENABLED` | `false` | No | Enables metric recording and `/metrics`. | `METRICS_ENABLED=true` |
| Diagnostics | `DIAG_ENABLED` | `true` | No | Starts periodic runtime snapshots; false disables the watcher. | `DIAG_ENABLED=false` |
| Signature | `AUTH_SIGNATURE_MASTER_KEY` | empty | No | Nonempty value enables HTTP signature validation. Secret value. | `AUTH_SIGNATURE_MASTER_KEY=<secret>` |
| Redis | `REDIS_ADDRESS` | empty | No | Presence enables Redis; value is the server address. | `REDIS_ADDRESS=127.0.0.1:6379` |
| Redis | `REDIS_PASSWORD` | empty | No | Redis password when authentication is enabled. Secret value. | `REDIS_PASSWORD=<secret>` |
| Memcached | `MEMCACHED_SERVERS` | empty | No | Presence enables Memcached; comma-separated servers. | `MEMCACHED_SERVERS=127.0.0.1:11211` |
| Kafka | `KAFKA_BROKERS` | empty | No | Presence enables Kafka; comma-separated brokers. | `KAFKA_BROKERS=127.0.0.1:9092` |
| Kafka/SASL | `KAFKA_USERNAME` | empty | Conditional | SASL username; requires password and mechanism. | `KAFKA_USERNAME=app-a` |
| Kafka/SASL | `KAFKA_PASSWORD` | empty | Conditional | Required with username. Secret value. | `KAFKA_PASSWORD=<secret>` |
| Kafka/SASL | `KAFKA_SASL_MECHANISM` | empty | Conditional | Supported values: `PLAIN`, `SCRAM-SHA-256`, `SCRAM-SHA-512`; required with username. | `KAFKA_SASL_MECHANISM=SCRAM-SHA-256` |
| Kafka/TLS | `KAFKA_JKS_FILE` | empty | Conditional | JKS or PEM trust material path; required if JKS password is set. | `KAFKA_JKS_FILE=/run/secrets/kafka-trust.jks` |
| Kafka/TLS | `KAFKA_JKS_PASSWORD` | empty | No | Password for JKS material; unused for PEM. Secret value. | `KAFKA_JKS_PASSWORD=<secret>` |
| Database | `DB_LIST` | empty | No | Comma-separated aliases; each alias activates its DB config. | `DB_LIST=primary,archive` |
| Database | `DB_<ALIAS>_DRIVER` | empty | For every listed alias | Must be `sqlserver` or `mysql`. | `DB_PRIMARY_DRIVER=mysql` |
| Database | `DB_<ALIAS>_HOST` | empty | For every listed alias | Host used to compose the driver DSN. | `DB_PRIMARY_HOST=127.0.0.1` |
| Database | `DB_<ALIAS>_PORT` | `0` | For every listed alias | Positive port used to compose the driver DSN. | `DB_PRIMARY_PORT=3306` |
| Database | `DB_<ALIAS>_NAME` | empty | For every listed alias | Database name used to compose the driver DSN. | `DB_PRIMARY_NAME=catalog` |
| Database | `DB_<ALIAS>_USER` | empty | For every listed alias | Database user used to compose the driver DSN. | `DB_PRIMARY_USER=app` |
| Database | `DB_<ALIAS>_PASSWORD` | empty | No | Database password. Secret value. | `DB_PRIMARY_PASSWORD=<secret>` |
| Database | `DB_<ALIAS>_PARAMS` | empty | No | Driver query parameters. | `DB_PRIMARY_PARAMS=parseTime=true` |

Each listed DB alias requires driver, host, positive port, name, and user; the loader composes the DSN. There is no active `DB_<ALIAS>_DSN` environment input. A service that constructs a `DBConfig` directly may set its `DSN` field. Alias text is uppercased for env names and lowercased in the Config map.

## Code Configuration Fields

The table lists all **100 fields** declared in `Config` and its nested configuration types: 84 leaf settings, 14 top-level group/derived fields, and the two nested fields in `AuthConfig`. Group rows show their Go type and point to the child settings below them.

Defaults below are assigned by `config.Load` / database alias loading. A service may override them in code before validation and app construction.

| Config path | Go type | Default after `Load` | Purpose |
|---|---|---|---|
| `App` | `AppConfig` | Values below | Application identity, environment, logging, and lifecycle timing. |
| `App.ServiceName` | `string` | `SERVICE_NAME` or empty | Service identity. Required by validation. |
| `App.Environment` | `string` | `APP_ENV` or empty | Environment name. Required by validation. |
| `App.LogLevel` | `string` | `LOG_LEVEL` or `info` | Logger threshold. |
| `App.DrainDelay` | `time.Duration` | `3s` | Readiness drain interval before component shutdown. |
| `App.ShutdownTimeout` | `time.Duration` | `25s` | App lifecycle timeout budget. |
| `Databases` | `map[string]DBConfig` | Empty map or aliases in `DB_LIST` | Database connections keyed by normalized alias. |
| `Databases[alias].Driver` | `string` | `DB_<ALIAS>_DRIVER`, lowercased | `sqlserver` or `mysql`. |
| `Databases[alias].DSN` | `string` | Composed by loader | The active loader composes this from driver/host/port/name/user/password/params. |
| `Databases[alias].Host` | `string` | `DB_<ALIAS>_HOST` or empty | Database host. |
| `Databases[alias].Port` | `int` | `DB_<ALIAS>_PORT` or `0` | Database port. |
| `Databases[alias].Name` | `string` | `DB_<ALIAS>_NAME` or empty | Database name. |
| `Databases[alias].User` | `string` | `DB_<ALIAS>_USER` or empty | Database username. |
| `Databases[alias].Password` | `string` | `DB_<ALIAS>_PASSWORD` or empty | Database password. |
| `Databases[alias].Params` | `string` | `DB_<ALIAS>_PARAMS` or empty | Driver query parameters. |
| `Databases[alias].Required` | `bool` | `true` | Whether database availability is required at startup/readiness. |
| `Databases[alias].MaxOpenConns` | `int` | `20` | SQL pool maximum open connections. |
| `Databases[alias].MaxIdleConns` | `int` | `10` | SQL pool maximum idle connections. |
| `Databases[alias].ConnMaxIdleTime` | `time.Duration` | `2m` | Maximum idle connection duration. |
| `Databases[alias].ConnMaxLifetime` | `time.Duration` | `5m` | Maximum connection lifetime. |
| `Migration` | `MigrationConfig` | Values below | Explicit Goose startup migration settings. |
| `Migration.AutoRun` | `bool` | `false` | Whether the service elects to run migrations at startup. |
| `Migration.DBName` | `string` | empty | Alias in `Databases` used by migration autorun. |
| `Migration.Dir` | `string` | empty | Migration file directory. |
| `Migration.LockEnabled` | `bool` | `true` | Use the migration lock when autorunning. |
| `Migration.LockKey` | `string` | `App.ServiceName` | Migration lock key. |
| `Migration.LockTimeout` | `time.Duration` | `30s` | Lock acquisition timeout. |
| `GRPC` | `GRPCConfig` | Values below | gRPC listener and TLS settings. |
| `GRPC.Port` | `int` | `GRPC_PORT` or `50051` | gRPC listener port. |
| `GRPC.TLSEnabled` | `bool` | `false` | Enable server TLS. |
| `GRPC.TLSCertFile` | `string` | empty | TLS certificate path, required if TLS enabled. |
| `GRPC.TLSKeyFile` | `string` | empty | TLS private-key path, required if TLS enabled. |
| `HTTP` | `HTTPConfig` | Values below | Gateway listener, middleware, and TLS settings. |
| `HTTP.Port` | `int` | `HTTP_PORT` or `8080` | HTTP gateway listener port. |
| `HTTP.TLSEnabled` | `bool` | `false` | Enable server TLS. |
| `HTTP.TLSCertFile` | `string` | empty | TLS certificate path, required if TLS enabled. |
| `HTTP.TLSKeyFile` | `string` | empty | TLS private-key path, required if TLS enabled. |
| `HTTP.PprofEnabled` | `bool` | `HTTP_PPROF_ENABLED` or `false` | Register pprof routes; forbidden in prod. |
| `HTTP.CORSAllowedOrigins` | `[]string` | `HTTP_CORS_ALLOWED_ORIGINS` or nil | Exact origins allowed by CORS. |
| `HTTP.CORSAllowedHeaders` | `[]string` | `Content-Type`, `Authorization`, `X-Request-ID` | Allowed request headers for CORS. |
| `HTTP.ReadHeaderTimeout` | `time.Duration` | `5s` | HTTP server header-read timeout. |
| `HTTP.ReadTimeout` | `time.Duration` | `15s` | HTTP server request-read timeout. |
| `HTTP.WriteTimeout` | `time.Duration` | `60s` | HTTP server response-write timeout; must exceed read timeout. |
| `HTTP.IdleTimeout` | `time.Duration` | `120s` | HTTP keep-alive idle timeout. |
| `HTTP.MaxBodyBytes` | `int64` | `1 MiB` | Request body limit; oversize maps to 413 / `CORE-1005`. |
| `HTTP.TrustedProxyHops` | `int` | `0` | Number of trusted reverse-proxy hops for client IP resolution. |
| `Observability` | `ObservabilityConfig` | Values below | OTLP trace export settings. |
| `Observability.OTLPEndpoint` | `string` | `APM_ENDPOINT` or empty | Nonempty endpoint enables OTLP tracing. |
| `Observability.OTLPInsecure` | `bool` | `false` | Use insecure OTLP transport; set only for trusted local networks. |
| `Observability.OTLPCACertFile` | `string` | empty | CA certificate path for OTLP TLS. |
| `Observability.TraceSamplingRatio` | `float64` | `0.1` | Trace sampling ratio. |
| `Metrics` | `MetricsConfig` | Values below | Metric recording gate. |
| `Metrics.Enabled` | `bool` | `METRICS_ENABLED` or `false` | Gate metric recording and `/metrics`. |
| `Log` | `LogConfig` | Values below | Structured logging and slow-operation settings. |
| `Log.MaskKeys` | `[]string` | `password`, `passwd`, `secret`, `token`, `authorization`, `apikey`, `api_key`, `pin`, `otp`, `cvv`, `card`, `private_key`, `session`, `phone`, `msisdn`, `account`, `cif`, `nik` | These keys are redacted from structured values; add service-specific keys in code. |
| `Log.RequestStart` | `bool` | `false` | Emit gRPC request-start debug logs. |
| `Log.SlowQueryThreshold` | `time.Duration` | `1s` | Threshold for slow database query logging. |
| `Log.SlowRequestThreshold` | `time.Duration` | `3s` | Threshold for slow request logging. |
| `Diagnostics` | `DiagnosticsConfig` | Values below | Runtime snapshot watcher settings. |
| `Diagnostics.Enabled` | `bool` | `DIAG_ENABLED` or `true` | Start periodic diagnostic watcher. |
| `Diagnostics.HeapThresholdPercent` | `int` | `80` | Trigger when the sampled heap-objects bytes reach this percentage of cgroup memory. |
| `Diagnostics.GoroutineThreshold` | `int` | `5000` | Goroutine-count trigger. |
| `Diagnostics.Cooldown` | `time.Duration` | `10m` | Minimum time between snapshots. |
| `Diagnostics.CheckInterval` | `time.Duration` | `30s` | Watcher sampling interval. |
| `Auth` | `AuthConfig` | Values below | Authentication settings. |
| `Auth.Signature` | `SignatureConfig` | Values below | HTTP request signature validation. |
| `Auth.Signature.Enabled` | `bool` | Master key is nonempty | Enable signature validation. |
| `Auth.Signature.MasterKey` | `string` | `AUTH_SIGNATURE_MASTER_KEY` or empty | HMAC secret. |
| `Auth.Signature.HeaderKey` | `string` | `x-signature` | Request signature header name. |
| `Auth.Signature.TimestampKey` | `string` | `x-timestamp` | Timestamp header name. |
| `Auth.Signature.MaxTimeDrift` | `time.Duration` | `5m` | Allowed timestamp drift. |
| `Auth.InternalJWT` | `InternalJWTConfig` | Values below | Internal JWT verification settings. |
| `Auth.InternalJWT.Enabled` | `bool` | `false` | Enable internal JWT verification. |
| `Auth.InternalJWT.PublicKey` | `string` | empty | RSA public key; required when enabled. |
| `Auth.InternalJWT.JWKSEndpoint` | `string` | empty | JWKS endpoint used by verifier configuration. |
| `Auth.InternalJWT.JWKSRefreshInterval` | `time.Duration` | `1h` | JWKS refresh interval. |
| `Auth.InternalJWT.Issuer` | `string` | empty | Expected issuer. |
| `Auth.InternalJWT.Audience` | `string` | empty | Expected audience. |
| `Auth.InternalJWT.Leeway` | `time.Duration` | `30s` | Token time-claim leeway. |
| `Auth.InternalJWT.IncludeMethods` | `[]string` | nil | Methods requiring auth when set. |
| `Auth.InternalJWT.ExcludeMethods` | `[]string` | nil | Methods excluded from auth; cannot combine with include list. |
| `Redis` | `RedisConfig` | Values below | Redis client settings. |
| `Redis.Enabled` | `bool` | `REDIS_ADDRESS` is present | Enable Redis. |
| `Redis.Address` | `string` | `REDIS_ADDRESS` or empty | Redis endpoint. |
| `Redis.Password` | `string` | `REDIS_PASSWORD` or empty | Redis password. |
| `Redis.DB` | `int` | `0` | Redis database number. |
| `Memcached` | `MemcachedConfig` | Values below | Memcached client settings. |
| `Memcached.Enabled` | `bool` | `MEMCACHED_SERVERS` is present | Enable Memcached. |
| `Memcached.Servers` | `[]string` | Comma-separated env or nil | Memcached endpoints. |
| `Memcached.Timeout` | `time.Duration` | `2s` | Memcached operation/health timeout. |
| `Kafka` | `KafkaConfig` | Values below | Kafka client/security settings. |
| `Kafka.Enabled` | `bool` | `KAFKA_BROKERS` is present | Enable Kafka integration. |
| `Kafka.Brokers` | `[]string` | Comma-separated env or nil | Kafka broker endpoints. |
| `Kafka.ClientID` | `string` | `App.ServiceName` | Kafka client identity. |
| `Kafka.Username` | `string` | `KAFKA_USERNAME` or empty | SASL username. |
| `Kafka.Password` | `string` | `KAFKA_PASSWORD` or empty | SASL password. |
| `Kafka.SASLMechanism` | `string` | `KAFKA_SASL_MECHANISM` or empty | SASL mechanism. |
| `Kafka.JksFile` | `string` | `KAFKA_JKS_FILE` or empty | JKS or PEM trust material path. |
| `Kafka.JksPassword` | `string` | `KAFKA_JKS_PASSWORD` or empty | JKS password. |
| `IgnoredEnvVars` | `[]string` | Sorted detected removed names | Loader-derived list of present removed env names; matching variables are ignored and logged once at startup. |

The table contains 86 leaf settings and the 14 aggregate/derived `Config` fields are listed above it. Environment loading reads 21 fixed names plus the seven database alias templates.

## Removed Environment Variable Names

These names no longer change runtime behavior. If present, they are collected in `Config.IgnoredEnvVars` and logged in one startup warning. Set the corresponding Config fields in code instead.

| Removed names or pattern | Use Config field |
|---|---|
| `LOG_TIMEZONE` | No replacement in Config; timestamps are UTC. |
| `SHUTDOWN_TIMEOUT` | `App.ShutdownTimeout` |
| `OTEL_EXPORTER_OTLP_ENDPOINT`, `OTEL_EXPORTER_OTLP_INSECURE`, `OTEL_EXPORTER_OTLP_CA_CERT_FILE`, `TRACE_SAMPLING_RATIO` | `Observability.OTLPEndpoint`, `OTLPInsecure`, `OTLPCACertFile`, `TraceSamplingRatio` |
| `REDIS_ENABLED`, `REDIS_DB` | `Redis.Enabled`, `Redis.DB` |
| `MEMCACHED_ENABLED`, `MEMCACHED_ADDRESS`, `MEMCACHED_TIMEOUT`, `MEMCACHE_HOST`, `MEMCACHE_PORT` | `Memcached.Enabled`, `Servers`, `Timeout` |
| `KAFKA_ENABLED`, `KAFKA_CLIENT_ID` | `Kafka.Enabled`, `ClientID` |
| `AUTH_SIGNATURE_ENABLED`, `AUTH_SIGNATURE_HEADER_KEY`, `AUTH_SIGNATURE_TIMESTAMP_KEY`, `AUTH_SIGNATURE_MAX_TIME_DRIFT` | `Auth.Signature.Enabled`, `HeaderKey`, `TimestampKey`, `MaxTimeDrift` |
| Prefixes `GRPC_TLS_*`, `HTTP_TLS_*`, `MIGRATION_*`, `INTERNAL_JWT_*` | Matching `GRPC`, `HTTP`, `Migration`, or `Auth.InternalJWT` fields |
| `DB_*_DSN`, `DB_*_REQUIRED`, `DB_*_MAX_OPEN_CONNS`, `DB_*_MAX_IDLE_CONNS`, `DB_*_CONN_MAX_IDLE_TIME`, `DB_*_CONN_MAX_LIFETIME` | Matching `Databases[alias]` fields; DSN is composed by loader |

## Local `.env`

```dotenv
SERVICE_NAME=catalog-api
APP_ENV=local
LOG_LEVEL=debug
GRPC_PORT=50051
HTTP_PORT=8080
METRICS_ENABLED=false
DIAG_ENABLED=true
```

The complete safe starter file is [`.env.example`](../.env.example). Optional integrations stay disabled unless their enabling address/list is set.

## Code Overrides

```go
cfg, err := config.Load(config.WithDotEnv(".env"))
if err != nil {
	return err
}
cfg.HTTP.ReadTimeout = 20 * time.Second
cfg.HTTP.WriteTimeout = 45 * time.Second
cfg.Metrics.Enabled = true
if err := cfg.Validate(); err != nil {
	return err
}
```

This is an excerpt for a function that returns `error`. The application logs an `effective_config` record at startup with sensitive values redacted. Avoid printing the Config struct yourself.