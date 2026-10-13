# grpcclient

`grpcclient` wraps a `grpc.ClientConn` with request identity propagation, OpenTelemetry client stats, a default per-call deadline, keepalive parameters, and service logging. Metrics are disabled unless `grpcclient.WithMetricsEnabled(true)` is supplied. Service-config retries are disabled.

```go
client, err := grpcclient.NewClient("payments.internal:50051", log)
if err != nil {
    return err
}
defer client.Close()

err = client.Conn().Invoke(ctx, "/payments.Service/Pay", request, response)
```

The default transport is plaintext to match go-core's default gRPC server configuration, where TLS is disabled. Configure transport credentials explicitly when the target uses TLS:

```go
import (
    "crypto/tls"

    "google.golang.org/grpc/credentials"
)

client, err := grpcclient.NewClient(
    "payments.internal:50051",
    log,
    grpcclient.WithTransportCredentials(credentials.NewTLS(tlsConfig)),
)
```

Calls without a context deadline receive a 30-second deadline. Existing deadlines are preserved. The client forwards request ID, user ID, and registered propagated labels from the context field bag.

`ResponseCodeFromHeader` reads `x-response-code` from gRPC response headers. `ErrorInfo` extracts the registered response code, class, finality, and retryability from a go-core gRPC status detail.