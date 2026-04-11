# Migration Guide

This file tracks consumer-visible upgrade changes for `go-core`.
Starting at `v1.0.0`, it is the canonical upgrade note for semver-governed releases.

Add an entry here when a change affects how a consuming service upgrades, including:

- exported API changes
- config or env contract changes
- runtime behavior changes that require service adjustment
- transport/auth/error behavior changes visible to consumers
- observability contract changes that affect dashboards or alerts

Do not add entries for internal-only cleanup with no upgrade impact.

## v1 baseline

`v1.0.0` is the first stable compatibility baseline for downstream adopters.
Entries below capture consumer-visible changes that landed before the first stable tag and therefore define the upgrade surface into `v1`.

## 1) `app.New` now requires context

Before:

```go
application, err := app.New(cfg)
```

After:

```go
application, err := app.New(ctx, cfg)
```

## 2) Deprecated app cache getter removed

Removed:

```go
application.Cache()
```

Use explicit cache selection:

```go
application.RedisCache()
application.MemcachedCache()
```

## 3) Cache interface `Get` now returns bytes

Before:

```go
value, err := cacheClient.Get(ctx, key) // string
```

After:

```go
value, err := cacheClient.Get(ctx, key) // []byte
```

## 4) Security claims are now generic only

Removed legacy fields from `security.Claims`:
- `UserID`
- `AccountNo`
- `Phone`
- `CIFCode`

Use:
- `Subject`
- `SessionID`
- `Role`
- `Attributes`

## 5) Metadata extractor header contract changed

When `INTERNAL_JWT_ENABLED=false`, supported headers are now:
- `x-subject`
- `x-session-id`
- `x-role`
- `x-claim-<name>`

Legacy headers (`x-user-id`, `x-user-role`, `x-session-*`) are no longer mapped.

## 6) JWT claim mapping tightened

Mapped claims:
- `sub` -> `Claims.Subject`
- `session_id`/`sid` -> `Claims.SessionID`
- `role` -> `Claims.Role`
- `attributes` object -> `Claims.Attributes`

Legacy fallback mapping from `user_id`, `uid`, `account_no`, etc. has been removed.

## 7) Outbox publisher option rename finalized

Removed:

```go
outbox.WithDriver("mysql")
```

Use:

```go
outbox.WithPublisherDriver("mysql")
```

## 8) Logger timezone behavior changed

Previously logger timestamps were hardcoded to `Asia/Jakarta`.

Now:
- default timezone is `UTC`
- optional env var `LOG_TIMEZONE` (IANA name) can override it.

## 9) Startup readiness helper signature changed

Before:

```go
server.LogStartupReadiness(ctx, log, grpcPort, httpPort, timeout)
```

After:

```go
server.LogStartupReadiness(ctx, log, grpcPort, httpPort, timeout, httpTLSEnabled)
```

## 10) OTLP transport is TLS-first by default

Before:
- OTLP exporter always used insecure transport.

Now:
- TLS is default for OTLP exporter.
- To keep insecure transport (local/dev only), set:
  - `OTEL_EXPORTER_OTLP_INSECURE=true`
- For custom CA, set:
  - `OTEL_EXPORTER_OTLP_CA_CERT_FILE=/path/to/ca.pem`

## 11) Migration autorun now has an additive logger-aware variant

Existing explicit startup migration remains valid:

```go
if err := migration.AutoRunUp(cfg); err != nil {
    return err
}
```

You may now opt into runtime signals without changing migration ownership:

```go
if err := migration.AutoRunUpWithLogger(cfg, application.Logger()); err != nil {
    return err
}
```

Behavior notes:
- startup migration remains explicit and service-owned
- lock safety semantics are unchanged
- logger-aware autorun emits `migration_autorun` and `migration_lock` `ServiceLog`

## 12) Database initialization now includes GORM

`database.New` now returns a wrapper that includes a GORM instance (`*gorm.DB`).

Before:
- Only raw `*sql.DB` was supported.

Now:
```go
db, err := database.New(cfg, log)
gormInstance := db.Gorm() // Access the pre-initialized GORM DB
```

GORM is initialized with `PrepareStmt: true` and `SingularTable: true` by default.

## 13) Enhanced Kafka configuration for SASL/JKS support

Kafka configuration has been expanded to support SASL Plain authentication and JKS certificates.

New environment variables:
- `KAFKA_USERNAME`
- `KAFKA_PASSWORD`
- `KAFKA_JKS_FILE`
- `KAFKA_JKS_PASSWORD`

These are optional and only used when `KAFKA_ENABLED=true`.
