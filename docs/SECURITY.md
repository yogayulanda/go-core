# Security

Auth modes:
- metadata extraction only (`INTERNAL_JWT_ENABLED=false`)
- JWT verification (`INTERNAL_JWT_ENABLED=true`)

Metadata contract (`INTERNAL_JWT_ENABLED=false`):
- `x-subject`
- `x-session-id`
- `x-role`
- `x-claim-<name>` => `security.Claims.Attributes["<name>"]`

JWT checks:
- signature + time claims
- optional issuer/audience
- explicit include/exclude method policy

Operational guidance:
- gRPC startup emits `auth_config` so operators can see whether the service is in metadata extraction mode or JWT verification mode
- JWT auth failures are sanitized to clients as unauthorized responses
- internal service logs record stable auth failure reasons such as missing authorization header, invalid token, invalid issuer, or invalid audience
- metadata extraction mode remains non-enforcing and is intended for trusted internal metadata propagation only

Claims contract:
- `Subject`
- `SessionID`
- `Role`
- `Attributes`

Never expose secrets or raw tokens in logs/errors.
