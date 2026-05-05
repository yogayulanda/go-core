# go-core · Security

## Auth Flow Overview

```
Incoming Request
      │
      ├─ [HTTP] → gateway middleware
      │           └─ Signature validation (if AUTH_SIGNATURE_ENABLED=true)
      │               ├─ HMAC-SHA256 of body using MASTER_KEY
      │               ├─ Timestamp drift check (MAX_TIME_DRIFT)
      │               └─ Reject → 401 if invalid
      │
      └─ [gRPC] → auth interceptor
                  └─ JWT enabled? (INTERNAL_JWT_ENABLED)
                      ├─ YES → JWT verification path
                      │         ├─ Extract Bearer token from metadata
                      │         ├─ Validate RS256/RS384/RS512 signature
                      │         ├─ Validate exp, nbf, iat
                      │         ├─ Validate issuer (if configured)
                      │         ├─ Validate audience (if configured)
                      │         ├─ Check include/exclude method policy
                      │         └─ Inject Claims into context
                      │
                      └─ NO → Metadata extraction (non-enforcing)
                              ├─ x-subject → Claims.Subject
                              ├─ x-session-id → Claims.SessionID
                              ├─ x-role → Claims.Role
                              └─ x-claim-<name> → Claims.Attributes["<name>"]
```

---

## JWT Verification Details

**Supported algorithms:** `RS256`, `RS384`, `RS512` (asymmetric only — symmetric HS* is blocked)

**Key sources (mutually exclusive, JWKS takes priority):**
1. `INTERNAL_JWT_JWKS_ENDPOINT` — dynamic key refresh via MicahParks/keyfunc v3
2. `INTERNAL_JWT_PUBLIC_KEY` — static RSA public key (PEM string or file path)

**Claims extracted into `security.Claims`:**
```go
type Claims struct {
    Subject    string            // "sub"
    SessionID  string            // "session_id" or "sid"
    Role       string            // "role"
    Attributes map[string]string // "attributes" map
}
```

**Method policy (evaluated at request time):**
- `INTERNAL_JWT_INCLUDE_METHODS` — only these gRPC methods require JWT
- `INTERNAL_JWT_EXCLUDE_METHODS` — all methods except these require JWT
- If neither is set → all methods require JWT when enabled
- Include list takes precedence over exclude list

**Leeway:** Default 30s clock skew tolerance (configurable via `INTERNAL_JWT_LEEWAY`)

---

## HTTP Payload Signature (HMAC)

Enabled via `AUTH_SIGNATURE_ENABLED=true`.

- Signature computed as HMAC-SHA256 of request body using `AUTH_MASTER_KEY`
- Signature sent in header key `AUTH_HEADER_KEY` (configurable)
- Timestamp header `AUTH_TIMESTAMP_KEY` validated against `AUTH_MAX_TIME_DRIFT`
- Protects against replay attacks and payload tampering on HTTP gateway

---

## Data Protection & Redaction

**Auto-redacted log keys** (matched by substring, case-insensitive):

```
password, passwd, secret, token, authorization,
apikey, api_key, pin, otp, cvv, card, private_key
```

**Redaction strategy:** Last 2 characters shown, remainder masked with `*`
- Example: `"mysecrettoken"` → `"***********en"`
- Empty values → `"**"`

**Applies to:**
- `map[string]interface{}` fields in `ServiceLog`, `DBLog`, `TransactionLog`
- `map[string]string` in `Claims.Attributes`
- Recursive traversal of nested maps

**What is NEVER logged:**
- Raw JWT tokens
- Database passwords (DSN is masked in logs)
- Card numbers, CVV, full payment credentials
- Full request bodies

---

## Secrets Handling

| Secret | Storage | Access Pattern |
|---|---|---|
| JWT public key | Env var or file path | Loaded once at startup via `NewInternalJWTVerifier` |
| HMAC master key | Env var | Read from `config.SignatureConfig.MasterKey` |
| DB password | Env var | Injected into DSN at config load, masked in logs |
| Redis password | Env var | Passed to Redis client, never logged |
| Kafka SASL password | Env var | JKS or SASL plain, configurable via `KafkaConfig` |

**Rules:**
- All secrets arrive via environment variables — no hardcoded credentials
- Secrets are never returned in API responses or error messages
- `config.Validate()` checks for required secret fields and rejects missing values at startup
- JWKS endpoint preferred over static key for key rotation support

---

## Auth Error Handling

Auth failures are sanitized before reaching the client:

| Error Type | Client Response | Internal Log |
|---|---|---|
| Empty token | `UNAUTHORIZED` | `authorization_token_empty` |
| Invalid signature/expired | `UNAUTHORIZED` | `invalid_token` |
| Wrong issuer | `UNAUTHORIZED` | `invalid_token_issuer` |
| Wrong audience | `UNAUTHORIZED` | `invalid_token_audience` |

Auth error codes are exposed via `security.AuthErrorCode(err)` for structured logging.
The gRPC interceptor always logs a `grpc_request` `ServiceLog` with the `auth_error_code` field.

---

## RBAC

`go-core` provides the **infrastructure** for RBAC but does **not enforce** role-based policies.

- `Claims.Role` carries the role string extracted from the JWT `role` claim
- `Claims.Attributes` carries arbitrary key-value claims (e.g., `tenant_id`, `scope`)
- Consuming services implement their own authorization logic using the injected claims
- `go-core` does not implement permission checks or role hierarchies — those are service-owned concerns

---

## Security Startup Observability

On startup, the gRPC server emits an `auth_config` `ServiceLog` with:
```json
{
  "auth_mode": "jwt" | "metadata",
  "policy_mode": "all" | "include" | "exclude" | "metadata",
  "jwt_enabled": true | false,
  "issuer_set": true | false,
  "audience_set": true | false,
  "include_method_count": 0,
  "exclude_method_count": 0,
  "leeway_ms": 30000
}
```

This ensures operators can verify auth configuration at startup without inspecting environment variables.

---

## Security Anti-Patterns to Avoid

- ❌ Using `INTERNAL_JWT_ENABLED=false` (metadata mode) for public-facing services
- ❌ Logging `Claims` or raw token values directly
- ❌ Adding HS256/HS512 to `ValidMethods` — this breaks RSA key requirements
- ❌ Storing secrets in config files committed to source control
- ❌ Growing `Claims` top-level fields with product-specific data — use `Attributes` map
