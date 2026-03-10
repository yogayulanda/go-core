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

Claims contract:
- `Subject`
- `SessionID`
- `Role`
- `Attributes`

Never expose secrets or raw tokens in logs/errors.
