# go-core · Conventions

## Code Style

- **Target Go version:** `1.24`
- **Linter:** `golangci-lint` (see `.golangci.yml`)
- **Format:** `gofmt` / `goimports` — all files must be formatted before commit
- **Module path:** `github.com/yogayulanda/go-core`

---

## API Design Rules

### Function Signatures

```go
// ✅ Correct — ctx first, named return only for defer-pattern
func (r *repo) Save(ctx context.Context, data Domain) error

// ✅ Correct — options pattern for extensibility
func NewKafkaPublisher(cfg KafkaConfig, opts ...PublisherOption) (Publisher, error)

// ❌ Wrong — no ctx, business arg before infra
func Save(data Domain, ctx context.Context) error
```

- `ctx context.Context` is always the first parameter in runtime functions
- Options pattern (`...Option`) for extensible constructors
- Interfaces kept small — prefer 1–3 methods
- Avoid naked `bool` returns — use `(result, error)` or named types

### Error Handling

```go
// ✅ Canonical error for application errors
return errors.New(errors.CodeNotFound, "user not found")
return errors.Wrap(errors.CodeInternal, "payment failed", internalErr)
return errors.Validation("invalid input", errors.Detail{Field: "amount", Reason: "must be positive"})

// ✅ Internal errors wrapped with context
return fmt.Errorf("repo.Save: %w", err)

// ❌ Never expose internal error message to API clients
return status.Error(codes.Internal, err.Error()) // raw internals leak!
```

- Always use `errors.AppError` as the canonical error type for application-facing errors
- Wrap internal errors with `fmt.Errorf("...: %w", err)` for traceability
- Internal error detail stays in logs — **never** in API responses
- `unknown errors` (non-AppError) must be mapped to `INTERNAL_ERROR`

### Logging

```go
// ✅ Use structured log flavors — not raw log.Info
logger.LogService(ctx, logger.ServiceLog{
    Operation: "order_create",
    Status:    "success",
    DurationMs: time.Since(start).Milliseconds(),
})

// ❌ Raw string logging loses structure
log.Info(ctx, "order created successfully")
```

- Use `LogService` for standard service operations
- Use `LogDB` for database operation diagnostics
- Use `LogTransaction` only for business-level transaction monitoring
- Use `logger.Logger.Info/Warn/Error` for informational/diagnostic messages
- Never log raw JWT tokens, passwords, or card data

### Testing

```go
// ✅ Prefer focused unit tests with mocks
func TestSave(t *testing.T) {
    db, mock, _ := sqlmock.New()
    mock.ExpectExec("INSERT INTO ...").WillReturnResult(...)
    // test isolated behavior
}

// ✅ Table-driven tests for multiple cases
func TestValidate(t *testing.T) {
    cases := []struct{ ... }{ ... }
    for _, tc := range cases {
        t.Run(tc.name, func(t *testing.T) { ... })
    }
}
```

- Use `sqlmock` for DB isolation
- Use function overrides (not interface mocking) for simple cases
- Cover success path + all documented error paths for exported functions
- Tests must pass with `go test ./...` before work is considered done

---

## Configuration Conventions

- All config via **environment variables** — no YAML/JSON config files for runtime config
- Config struct fields use Go types (`time.Duration`, `int`, `bool`) — not raw strings
- `UPPERCASE_SNAKE_CASE` for all env var names
- DB alias in env: `UPPERCASE` → normalized to `lowercase` in code
- Validation in `config.Validate()` — fail fast at startup, never at request time
- Additive DX improvements go through `ValidateIssues()` — compact path unchanged

---

## Naming Conventions

| Element | Convention | Example |
|---|---|---|
| Package | lowercase single word | `security`, `resilience` |
| Exported type | PascalCase | `AppError`, `CircuitBreaker` |
| Interface | Noun or Noun+er | `Publisher`, `Consumer`, `Beginner` |
| Constructor | `New<Type>(...)` | `NewInternalJWTVerifier(...)` |
| Error variables | `err<Description>` | `errInvalidToken` |
| Metric names | `app_<category>_<measurement>_<unit>` | `app_request_duration_seconds` |
| Log operation names | `snake_case` verb+noun | `"payment_process"`, `"app_init"` |
| Log status values | stable lowercase | `"success"`, `"failed"`, `"pending"` |

---

## Lifecycle Ownership Rules

```go
// ✅ All shutdown hooks must be explicit
lifecycle.Register(func(ctx context.Context) error {
    return db.Close()
})

// ❌ Hidden background goroutines not controlled by consuming service
go func() { worker.Start(ctx) }() // inside go-core — forbidden
```

- `go-core` never starts background goroutines automatically
- All cleanup must be registered in `lifecycle`
- Consuming service decides: which workers run, in which pods, at what interval

---

## Boundary Enforcement

### Allowed in `go-core`

- Generic infrastructure helpers (transport, logging, config, DB, cache, messaging)
- Technical contracts intentionally standardized across services (e.g., `TransactionLog`)
- Observability baselines (Prometheus metrics, OTEL tracing)

### Not Allowed in `go-core`

- Business entities, domain models, product schemas
- Service-specific DB alias defaults (e.g., never hardcode `"transaction"` as a DB name)
- Product-specific event payloads or topic names
- Hidden background behavior not controllable by consuming service
- Generic utilities better suited for `utils-shared`

---

## Change Checklist

Before any change to a high-risk area (`app/`, `config/`, `server/`, `errors/`, `security/`):

1. **Compatibility** — Does this break existing consuming services?
2. **Coupling** — Does this introduce product-specific knowledge?
3. **Concurrency** — Are there new goroutines or shared state?
4. **Scale risk** — Does this cause metric cardinality explosion or connection growth?
5. **Overengineering** — Is this simpler than the problem requires?

For any public-contract change:
- Update `README.md`
- Update relevant `docs/` file
- Update `MIGRATION.md`
- Update `.ai/` context if behavior changes
- Run `go test ./...` and `make quality-gate`

---

## Semver and Release Rules

- `v1.x.y` is the stable series — published public API is a compatibility contract
- **Patch** (`y`): bug fixes, internal refactors, no API change
- **Minor** (`x`): additive new exported API, no breaking changes
- **Major** (new `v2`): breaking change to any public API, config, or runtime behavior

Breaking changes require:
- Explicit major-version decision
- Entry in `MIGRATION.md`
- Upstream team communication

---

## Task Definition Standard

All `.ai/tasks/*.md` files must define:
```yaml
goal:        [one-line intent]
scope:       [layer(s) affected]
allowed_paths: [list of permitted file paths]
constraints: [what must not change]
```

AI must not implement behavior outside `allowed_paths`.
Work is complete only when: implementation done + tests pass + docs aligned + `.ai` context updated.
