Code Ownership Map

`go-core` owns:

- `app/*`
- `config/*`
- `database/*`
- `dbtx/*`
- `migration/*`
- `server/*`
- `cache/*`
- `messaging/*`
- `observability/*`
- `security/*`
- `errors/*`
- `logger/*`
- `resilience/*`
- `templates/*`
- `examples/*`
- `version/*`
- `docs/*`

Boundary exclusions:

- service-specific DB aliases
- business workflow semantics
- product-specific field contracts
- generic utilities better housed in `utils-shared`

Allowed shared technical exception:

- selected platform-standard technical contracts intentionally shared across services, such as transaction observability

When a change is better implemented in a consuming service, keep it out of this repo.
