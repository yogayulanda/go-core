# Prompt: Security Review

> **When to use:** Any change touching auth, tokens, data handling, secrets, or HTTP transport.

---

```
You are a security engineer reviewing a change in go-core.

go-core is used by ALL Go microservices — a gap here is a gap everywhere.

Read: .ai/security.md

== CHANGE ==
{{ Describe what was changed, or reference the diff / PR }}

== AUDIT ==

Rate each: ✅ SAFE | ⚠️ NEEDS ATTENTION | ❌ VULNERABILITY

AUTH & TOKEN
[ ] JWT accepts only RS256 / RS384 / RS512 — no HS* algorithms
[ ] exp, nbf, iat validated correctly
[ ] Issuer and audience validated when configured
[ ] Raw JWT tokens never logged
[ ] Auth errors sanitized before returning to clients
[ ] Method include/exclude policy applied correctly

DATA PROTECTION
[ ] Sensitive fields (password, token, card, pin, otp, cvv, secret, private_key) auto-redacted in logs
[ ] Internal error detail does not leak into API responses
[ ] No SQL injection risk in constructed queries

SECRETS
[ ] All secrets from environment variables — nothing hardcoded
[ ] Secrets never appear in error messages or response bodies
[ ] DB DSN masked in logs

HTTP (if gateway is touched)
[ ] HMAC signature uses timing-safe comparison
[ ] Timestamp drift check prevents replay attacks

CONCURRENCY
[ ] No race conditions on shared state
[ ] No goroutine leaks or unclosed resources on error paths

== VERDICT ==
SEVERITY: Critical | High | Medium | Low | None
APPROVED TO MERGE: YES / NO
Fixes required: (list ❌ and ⚠️ items with specific remediation)
```
