# Migration Guide

This file lists breaking API changes made in the latest cleanup pass.

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
