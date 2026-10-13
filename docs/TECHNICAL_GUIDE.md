# Technical Guide

Panduan ini menjelaskan API yang tersedia pada source saat ini. Integrasi internal 1.0.0 yang dijelaskan di sini belum ditag; lihat `MIGRATION.md` dan catatan blockers.

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

`grpcclient.NewClient(target, log, options...)` provides `Conn()`, `Close()`, `Dial(...)`, identity metadata propagation, OTel stats, call logs/metrics, keepalive, and a 30-second default deadline when the caller supplied none. Retries are disabled, including service-config retries. The default transport is plaintext to match the default server; supply `WithTransportCredentials(...)` for TLS. `ResponseCodeFromHeader` and `ErrorInfo` read go-core response metadata.

## Kafka, Outbox, And Audit Events

Kafka supports SASL `PLAIN`, `SCRAM-SHA-256`, and `SCRAM-SHA-512`. The truststore loader autodetects JKS or PEM and returns a TLS config used by writers, readers, and readiness checks.

Consumer concurrency is keyed by partition: a partition has one FIFO worker, and a failed offset blocks later offsets unless a successful DLQ publication allows it to commit. DLQ records preserve headers and add `x-dlq-reason` and `x-original-topic`.

`messaging/outbox.Publisher.PublishTx` stores a UUID `event_id` with identity row `id`. A worker claims per key under a short lease transaction, publishes outside it, and stores `SENT`, retry/backoff, or `DEAD`. DDL is in `docs/MESSAGING_PATTERN.md`.

`outbox.Publisher.PublishAudit` validates CloudEvents 1.0 including `ce_sequence`; the subject is the Kafka key and `ce_id` is the outbox ID. It logs metadata only. Consumers can access restored headers with `messaging.CloudEventsFromContext(ctx)`.

## Diagnostics And Release Gaps

`DIAG_ENABLED=true` enables bounded runtime diagnostics (goroutine threshold 5000, heap threshold 80% of cgroup memory, 10m cooldown, 30s check interval). Snapshots contain function/file locations and numeric stats only.

Still undecided/out of scope: `GC-23` tracing backend, `GC-24` metrics policy, `D-PATH` internal module path, and `Q-SIG` signature scheme confirmation. No module path change or tag is included here.
