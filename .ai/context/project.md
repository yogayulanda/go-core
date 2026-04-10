Project: go-core

Purpose:
Reusable infrastructure foundation for Go services.
This repo acts as the standard foundation used to bootstrap and run service applications.

Primary goals:

- stable bootstrap/runtime behavior
- generic environment-driven configuration
- transport and observability baseline
- reusable infra helpers without product coupling
- foundation-first composition for service apps, not utility aggregation
- room for selected platform-standard technical contracts when intentionally standardized

Module scope:

- `app`, `config`, `server`, `server/grpc`, `server/gateway`
- `database`, `dbtx`, `migration`
- `cache`, `messaging`, `messaging/outbox`
- `observability`, `logger`, `security`, `resilience`, `errors`
- `templates`, `examples`, `version`, `docs`
