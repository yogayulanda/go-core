# httpclient

`httpclient` provides a resilient and instrumented HTTP client wrapper for `go-core` services. It is based on `resty` and integrates seamlessly with the framework's logging, tracing, and resilience features.

## Features

- **Resilience**: Integrated with `sony/gobreaker` for circuit breaking and configurable retries with backoff.
- **Observability**: Automatically propagates OpenTelemetry spans and emits structured `ServiceLog`.
- **Sensible Defaults**: Timeout, retry, and circuit breaker policies come with pre-configured defaults aligned with `go-core` standards.

## Usage

### Basic Initialization

```go
import (
    "github.com/yogayulanda/go-core/httpclient"
    "github.com/yogayulanda/go-core/logger"
)

// ... inside your application bootstrap
log, _ := logger.New("my-service", "info")
client := httpclient.NewClient(log)
```

### Performing Requests

Use the `Do` method to execute requests with context propagation and circuit breaker protection:

```go
req := client.Request().SetBody(myPayload)
resp, err := client.Do(ctx, req, http.MethodPost, "https://api.external.com/v1/data")
```

Or use convenience methods:

```go
resp, err := client.Get(ctx, "https://api.external.com/v1/health")
```

### Advanced Configuration

```go
client := httpclient.NewClient(log,
    httpclient.WithTimeout(10 * time.Second),
    httpclient.WithUserAgent("my-custom-agent/1.0"),
    httpclient.WithoutTracing(),
)
```

## Resilience Details

### Retries
By default, the client retries on 5xx status codes and 429 (Too Many Requests). You can customize this via `WithRetryOptions`.

### Circuit Breaker
The default circuit breaker uses `sony/gobreaker` and triggers on 5xx responses. When the circuit is open, requests fail fast with an error, preventing downstream pressure on failing external services.

## Logging

Each request and response (or error) emits a `ServiceLog` with:
- `operation`: `httpclient_request` / `httpclient_response`
- `metadata`: method, URL, status code, and duration.
