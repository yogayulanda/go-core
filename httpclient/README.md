# httpclient

`httpclient` provides a resilient and instrumented HTTP client wrapper for `go-core` services. It is based on `resty` and integrates seamlessly with the framework's logging, tracing, and resilience features.

## Features

- **Resilience**: Integrated with `sony/gobreaker` for circuit breaking; retries are opt-in.
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
    httpclient.WithIdleConnTimeout(30 * time.Second),
    httpclient.WithUserAgent("my-custom-agent/1.0"),
    httpclient.WithoutTracing(),
)
```

Retries are disabled by default. Enable them explicitly for a client with `WithRetry`:

```go
import "github.com/yogayulanda/go-core/resilience"

client := httpclient.NewClient(log,
    httpclient.WithRetry(resilience.DefaultRetryOptions()),
)
```

HTTP status responses, including 5xx responses, are returned with a nil error. `TransportError` describes failures before or after request headers were sent. Identity headers are forwarded only when `WithInternalPropagation()` is enabled.

Request and response bodies are never logged by default. To opt in, provide a masker with `WithBodyLogging`; the callback is applied to both bodies before either is added to call metadata:

```go
httpclient.WithBodyLogging(func(body []byte) []byte {
    return []byte("[masked]")
})
```

## Resilience Details

### Retries
The client does not retry by default. `WithRetry` accepts the existing resilience retry policy when retries are appropriate for the target operation.

### Circuit Breaker
The default circuit breaker uses `sony/gobreaker` and triggers on 5xx responses. When the circuit is open, requests fail fast with an error, preventing downstream pressure on failing external services.

## Logging

Each call emits one `ServiceLog` with operation `httpclient_call`. Metadata includes method, sanitized URL, status code when a response was received, duration, and transport phase/kind on transport failures.
