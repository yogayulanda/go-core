# Event Contract

Use `messaging.Message`:
- `Topic`
- `Key`
- `Payload`
- `Headers`

Payload envelope should include:
`event_id`, `event_type`, `event_version`, `occurred_at`.

Prefer additive event evolution and idempotent consumers.
