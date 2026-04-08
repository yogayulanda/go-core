Project: go-core

Purpose:
Reusable infrastructure foundation for Go services.
This repo should act as the standard foundation used to bootstrap and run service applications.
It is not the place for generic utility accumulation.

Primary goals:

- stable bootstrap/runtime behavior
- generic environment-driven configuration
- transport and observability baseline
- reusable infra helpers without product coupling
- foundation-first composition for service apps, not utility aggregation
- room for selected platform-standard technical contracts when intentionally standardized

Current module scope:

- `app`, `config`, `server`, `server/grpc`, `server/gateway`
- `database`, `dbtx`, `migration`
- `cache`, `messaging`, `messaging/outbox`
- `observability`, `logger`, `security`, `resilience`, `errors`
- `templates`, `examples`, `version`, `docs`

Current adoption stage:

- actively shaped around real usage in `transaction-history-service`
- still early enough to permit bounded refactors
- should now converge toward a stable standard foundation for future Go services

Important design rule:

If behavior can reasonably live in a service repository, keep it out of `go-core`.

Companion boundary:

If code is merely a generic helper and does not strengthen the application foundation, it belongs in `utils-shared` instead.
