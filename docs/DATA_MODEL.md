# Data Model

`go-core` owns technical models only:
- `config.Config`
- `errors.AppError` / `errors.ErrorResponse`
- `messaging.Message`
- `outbox.Event`

Business tables/entities stay in service repo.
