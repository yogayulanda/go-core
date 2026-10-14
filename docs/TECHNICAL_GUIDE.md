# Technical Guide

Panduan ini menjelaskan API yang tersedia pada source v1.1.0. Untuk perubahan dari v1.0.x, lihat `MIGRATION.md`.

## Configuration And Startup

`config.Load` membaca environment yang didukung dan `cfg.Validate()` memeriksa kombinasi nilainya. `SERVICE_NAME` dan `APP_ENV` wajib. Contoh environment yang aman ada di `.env.example`.

```go
cfg, err := config.Load(config.WithDotEnv(".env"))
if err != nil {
    return err
}
if err := cfg.Validate(); err != nil {
    return err
}
```

`migration.AutoRunUp(cfg)` adalah keputusan eksplisit milik service; `app.New` tidak menjalankan migrasi. Demikian pula, outbox worker tidak mulai otomatis.

## HTTP And gRPC Servers

Buat server melalui `server/grpc.New(application)` dan `server/gateway.New(application, registerFunc)`. Endpoint gRPC-Gateway harus mengarah ke server gRPC lokal (`RegisterXHandlerFromEndpoint`) supaya interceptor tetap dilewati.

Gateway memasang `/health`, `/ready`, `/version`, `/metrics`, CORS allowlist, body limit, configured timeouts, request identity, and response envelopes. Successful API responses include `status`, `response_code`, `message`, `request_id`, `timestamp`, and `data`; failures use the same common fields and validation may include `details`. Operational endpoints bypass the API envelope. `CORE-1005` maps an oversized body to HTTP 413 / `ResourceExhausted`.

Saat shutdown, `/ready` menjadi 503 dan gRPC health menjadi `NOT_SERVING`, kemudian `DrainDelay` (3s default) berjalan sebelum server dihentikan graceful di bawah `ShutdownTimeout` (25s default). `ShutdownTimeout + DrainDelay` sebaiknya muat dalam termination grace period.

### Raw Gateway Responses

Route gateway normal membungkus response 2xx dengan success envelope go-core. Gunakan `gateway.Raw` untuk route yang harus meneruskan response transport apa adanya, misalnya transparent proxy, webhook callback, atau file download:

```go
err := mux.HandlePath(http.MethodGet, "/partner/{id}", gateway.Raw(func(w http.ResponseWriter, r *http.Request, params map[string]string) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusOK)
    _, _ = w.Write([]byte(`{"partner":"response"}`))
}))
```

Untuk handler biasa yang mengimplementasikan `http.Handler`, bungkus dengan `gateway.RawHTTP(handler)` lalu panggil dari adapter `HandlePath`:

```go
err := mux.HandlePath(http.MethodGet, "/download", func(w http.ResponseWriter, r *http.Request, _ map[string]string) {
    gateway.RawHTTP(downloadHandler).ServeHTTP(w, r)
})
```

Raw melewati success envelope saja; request identity, panic recovery, body limit, signature validation, tracing, CORS, metrics, dan access logging tetap berlaku. Handler bertanggung jawab penuh atas status, headers, content type, body, serta format error response. Jangan gunakan Raw untuk API normal yang mengandalkan envelope go-core. Untuk menemukan pendaftaran raw di source service, jalankan `rg -n 'gateway\.Raw(HTTP)?\('`.

## Database And Transactions

SQL driver yang didukung: `sqlserver` dan `mysql`. `database.New(cfg, log, database.WithSlowQueryThreshold(time.Second))` mengembalikan wrapper yang menyediakan raw `*sql.DB` dan `Gorm()`.

Gunakan `dbtx.WithTx` pada batas use case dan `dbtx.FromContext` di repository. Untuk outbox, tulis event di transaksi bisnis yang sama, lalu biarkan worker service-owned menerbitkannya setelah commit.

## Errors And Envelopes

Daftarkan definisi kode aplikasi dengan `errors.Register`. Buat error menggunakan `errors.New`, `errors.Wrap`, atau `errors.Validation`. Registry menentukan class, finality, retryability, HTTP status, dan gRPC status. Kode tidak terdaftar dipetakan ke `CORE-9000`; validasi `CORE-1000` dapat membawa field details.

Gunakan `response.SetCode(ctx, code)` untuk mengirim response code dari handler gRPC kepada gateway. Gateway mengambil metadata itu dan tidak mengekspos header `x-response-code` ke klien HTTP.

## Logging And Identity

`logger.New(service, level, logger.WithEnvironment(...), logger.WithConfig(...))` membuat ECS JSON kecuali environment `local`, yang menggunakan console encoder. Database slow-query threshold dapat dikonfigurasi melalui `logger.Config` dan `database.WithSlowQueryThreshold`.

Gateway membuat request field bag dan menangkap request ID, user agent, serta client IP berdasarkan `TrustedProxyHops`. `observability.SetUserID`, `SetLabel`, dan `RegisterPropagatedLabel` menyediakan identity. Satu request REST yang mencapai gRPC mendapat satu access log dari gRPC interceptor; gateway menulis access log hanya untuk kegagalan sebelum gRPC.

## Outbound Clients

### HTTP

Retries off by default; opt in per client with `httpclient.WithRetry(resilience.DefaultRetryOptions())`. Every received HTTP response, including 5xx, is returned with `err == nil`. Transport failures are `*httpclient.TransportError{Phase, Kind, Err}`; phase reflects whether request headers were written. Logs contain a URL without query, fragment, or userinfo. Identity forwarding requires `httpclient.WithInternalPropagation()`. Body logging requires `WithBodyLogging(masker)` and no raw body is logged by default.

### gRPC

`grpcclient.NewClient(target, log, options...)` provides `Conn()`, `Close()`, `Dial(...)`, identity metadata propagation, OTel stats, call logs, keepalive, and a 30-second default deadline when the caller supplied none. Per-call metrics require `grpcclient.WithMetricsEnabled(true)`. Retries are disabled, including service-config retries. The default transport is plaintext to match the default server; supply `WithTransportCredentials(...)` for TLS. `ResponseCodeFromHeader` and `ErrorInfo` read go-core response metadata.

## Kafka, Outbox, And Audit Events

Kafka supports SASL `PLAIN`, `SCRAM-SHA-256`, and `SCRAM-SHA-512`. The truststore loader autodetects JKS or PEM and returns a TLS config used by writers, readers, and readiness checks.

Consumer concurrency is keyed by partition: a partition has one FIFO worker, and a failed offset blocks later offsets unless a successful DLQ publication allows it to commit. DLQ records preserve headers and add `x-dlq-reason` and `x-original-topic`.

`messaging/outbox.Publisher.PublishTx` stores a UUID `event_id` with identity row `id`. A worker claims per key under a short lease transaction, publishes outside it, and stores `SENT`, retry/backoff, or `DEAD`. DDL is in `docs/MESSAGING_PATTERN.md`.

`outbox.Publisher.PublishAudit` validates CloudEvents 1.0 including `ce_sequence`; the subject is the Kafka key and `ce_id` is the outbox ID. It logs metadata only. Consumers can access restored headers with `messaging.CloudEventsFromContext(ctx)`.

## Diagnostics And Release Gaps

`DIAG_ENABLED=true` enables bounded runtime diagnostics (goroutine threshold 5000, heap threshold 80% of cgroup memory, 10m cooldown, 30s check interval). Snapshots contain function/file locations and numeric stats only.

Tracing is OpenTelemetry OTLP to Elastic APM Server via `APM_ENDPOINT`; no Elastic APM agent is used and logs do not include `transaction.id`. Metrics are disabled by default; set `METRICS_ENABLED=true` to record metrics and register `/metrics`. `app_transaction_total` is removed.

Downstream forks may adjust the module path before adoption. `Q-SIG` confirmation is deferred but is not a release blocker. Consumers need Go 1.26.0 or newer; CI validates with Go 1.26.9. The rolling N-1 toolchain and security-patch policy is described in `MIGRATION.md`.

## Hot-Path Benchmark Method

Use this method when deciding whether a toolchain or dependency change regresses the gateway, interceptor, or outbox paths:

- Set `GOMAXPROCS=4` for every compared variant.
- Exclude tests with `-run='^$'`; use `-benchmem` and ten or more independent samples.
- Interleave variants to reduce drift, running each variant with `-count=1` in every round. The upgrade comparison used 20 rounds, each executing A (main + Go 1.24.13), B (main + Go 1.26.9), then C (branch + Go 1.26.9).
- Compare samples with `benchstat`, reporting latency, bytes, allocations, sample counts, and significance. Compare A to B for toolchain/runtime effects and B to C for dependency/code effects.
- A hot-path regression gate is valid only for measurements made with this locked-procs, interleaved method on a stable runner. WSL measurements are noisy and have roughly 10% practical detection limits; confirm borderline results on a stable Linux CI runner.

The recorded gateway measurement uses `httptest.NewRecorder` and does not include a TCP or grpc-gateway-to-gRPC proxy hop. Track that path separately using an in-process gRPC transport such as bufconn. Go 1.26.9 does not recognize `GOEXPERIMENT=nogreenteagc`, so the alternate-GC comparison is unavailable.
