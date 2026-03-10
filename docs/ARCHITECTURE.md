# Architecture

## Shape
- Core contracts: config, lifecycle, error, messaging interfaces.
- Core adapters: DB/cache/kafka/grpc/gateway.
- Service code: domain/usecase/repository/transport.

## Runtime Path
1. load config
2. build `app.App`
3. build grpc + gateway
4. run via `server.Run(...)`
5. graceful shutdown via lifecycle

## Extension Rule
Add only when reused by multiple services.
