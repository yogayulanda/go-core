# Panduan Teknis Lengkap `go-core`

Dokumen ini adalah ringkasan ramah-developer untuk memahami fitur apa saja yang dimiliki oleh `go-core`, apa yang bisa dikonfigurasi, dan bagaimana contoh implementasinya di *microservice* Anda.

---

## 1. Konfigurasi Sistem (`config/`)
Semua *service* dikendalikan penuh oleh **Environment Variables (`.env`)**. `go-core` otomatis memvalidasi apakah variabel yang diwajibkan sudah diisi sebelum aplikasi menyala (Fail-Fast).

**Variabel Utama:**
- `SERVICE_NAME` (Wajib)
- `APP_ENV` (Default: `dev`. Jika di-set `production`, log otomatis menjadi JSON tanpa warna)
- `LOG_LEVEL` (Default: `info`)

**Contoh Penggunaan:**
```go
cfg, err := config.Load(config.WithDotEnv(".env"))
if err := cfg.Validate(); err != nil {
    panic(err)
}
```

---

## 2. Jaringan & Server (`server/`)
`go-core` menggunakan pola **Multiplexing**. Ia menjalankan **gRPC Server** (untuk komunikasi antar-backend) dan **HTTP API Gateway** (untuk komunikasi Frontend/Mobile) dari **1 file definisi Protobuf saja**.

**Variabel Konfigurasi:**
- `GRPC_PORT` (Default: 50051)
- `HTTP_PORT` (Default: 8080)
- `HTTP_PPROF_ENABLED` (Default: false. Jika true, menyalakan `/debug/pprof` untuk mendeteksi kebocoran memori).
- `GRPC_TLS_ENABLED` & `HTTP_TLS_ENABLED` (Untuk koneksi aman via SSL/TLS).

**Fitur Otomatis Gateway HTTP:**
- **Success & Error Envelope:** Semua respons otomatis dibungkus dengan `{ "success": true/false, "trace_id": "...", "data": ... }`.
- **Panic Recovery:** Server tidak akan mati meskipun kode Anda mengalami `panic()`.
- **Health & Ready Endpoint:** Akses `/health` dan `/ready` otomatis terpasang.

---

## 3. Database & Transaksi (`database/` dan `dbtx/`)
Mendukung *koneksi jamak (Multiple Databases)* sekaligus dan menyediakan bantuan injeksi konteks agar kode transaksi tetap bersih.

**Variabel Konfigurasi:**
- `DB_LIST` (Contoh: `primary,ledger`. Menentukan nama alias database).
- `DB_PRIMARY_DRIVER`, `DB_PRIMARY_HOST`, `DB_PRIMARY_USER`, `DB_PRIMARY_PASSWORD` (Kredensial spesifik per alias).

**Contoh Implementasi Transaksi Aman (`dbtx`):**
```go
// Di dalam Usecase/Service
err := dbtx.WithTx(ctx, s.db, func(txCtx context.Context) error {
    // Parameter txCtx ini secara ajaib menyimpan koneksi transaksi SQL!
    repo.KurangiSaldo(txCtx, 1000)
    repo.TambahSaldo(txCtx, 1000)
    return nil // Otomatis Commit jika return nil, Otomatis Rollback jika error.
})
```

---

## 4. Keamanan & Autentikasi (`security/`)
Menyediakan pencegat (*Interceptor*) yang menolak otomatis akses jika Token JWT kedaluwarsa atau tidak valid, tanpa perlu Anda tulis ulang di setiap *service*.

**Variabel Konfigurasi:**
- `INTERNAL_JWT_ENABLED` (Jika true, semua endpoint gRPC akan memvalidasi Token).
- `INTERNAL_JWT_PUBLIC_KEY` (Kunci RSA Publik untuk mencocokkan *signature*).
- `INTERNAL_JWT_EXCLUDE_METHODS` (Contoh: `/user.Auth/Login`. Mengecualikan endpoint tertentu dari pengecekan JWT).
- `AUTH_SIGNATURE_ENABLED` (Jika true, mencegah serangan *Replay Attack* dari HTTP Gateway).

---

## 5. Observability & Logging (`logger/` & `observability/`)
Mengatur agar jejak pencarian eror (Trace) dapat dengan mudah dilakukan oleh *Customer Support* dan *DevOps*.

**Variabel Konfigurasi:**
- `OTEL_EXPORTER_OTLP_ENDPOINT` (Mengirim *trace* jarak jauh ke sistem seperti Jaeger/Datadog).

**Contoh Implementasi Logging:**
```go
// 1. Log Teknis (Untuk DevOps, tidak butuh struktur kaku)
s.log.Error(ctx, "koneksi database terputus", logger.Field{Key: "db", Value: "primary"})

// 2. Log Bisnis (Untuk Audit Compliance/Security)
s.log.LogEvent(ctx, logger.EventLog{
    Category: "security_audit",
    Action:   "reset_password",
    ActorID:  "user-999",
})

// 3. Log Keuangan Khusus Fintech (Untuk Analis Data / Finance)
s.log.LogTransaction(ctx, logger.TransactionLog{
    Operation: "topup_wallet",
    Status:    "success",
    Metadata:  map[string]interface{}{"amount": 50000},
})
```

---

## 6. Integrasi Eksternal (Kafka, Redis, Memcached)
Semua dependensi pihak ketiga dirancang bersifat **Opsional (Opt-In)**. Jika konfigurasi di-set `false`, `go-core` sama sekali tidak akan menelan memori untuk komponen tersebut.

**Variabel Konfigurasi:**
- `REDIS_ENABLED` & `REDIS_ADDRESS`
- `MEMCACHED_ENABLED` & `MEMCACHED_SERVERS`
- `KAFKA_ENABLED` & `KAFKA_BROKERS`

---

## 7. Penanganan Error Standar (`errors/`)
Ucapkan selamat tinggal pada error Go biasa `fmt.Errorf()`. Semua layanan mewajibkan tipe data *ErrorBuilder* terstruktur.

**Contoh Implementasi Error:**
```go
if nominal <= 0 {
    return coreErrors.Build("TRF", coreErrors.CategoryVAL, "001").
        Message("nominal transfer negatif").            // Log teknis rahasia (disembunyikan)
        UserMessage("Nominal transfer tidak valid!").   // Dikirim ke layar HP
        Finality(coreErrors.FinalityBusiness).          // Jangan di-retry
        Done()
}
```

---

## 8. Migrasi Skema Database Otomatis (`migration/`)
Digunakan untuk mengeksekusi perubahan tabel *database* (seperti `CREATE TABLE`, `ALTER`) saat aplikasi pertama kali menyala (menggunakan basis `goose`).

**Variabel Konfigurasi:**
- `MIGRATION_AUTO_RUN` (Jika `true`, aplikasi akan menjalankan migrasi sebelum server HTTP/gRPC diizinkan menyala).
- `MIGRATION_DB` (Menunjuk alias database mana yang akan dimigrasi, contoh: `primary`).
- `MIGRATION_DIR` (Folder tempat file SQL disimpan, misal: `migrations/primary`).
- `MIGRATION_LOCK_ENABLED` (Otomatis mencegah *race condition* jika aplikasi Anda di-deploy ke banyak K8s Pod sekaligus).

---

## 9. Pemanggilan Eksternal yang Kebal (*Resilience & HTTP Client*)
Ketika aplikasi Anda memanggil API pihak ketiga (misal: API Bank atau Payment Gateway), koneksi sangat rentan terhadap gangguan. `go-core` menyediakan modul `resilience/` dan `httpclient/` untuk melindunginya.

**Fitur HTTP Client (`httpclient/`):**
- **Otomatis Retry:** Jika API Bank mengembalikan HTTP 500, *client* akan mencoba ulang secara otomatis (dengan pola penundaan *Backoff*).
- **Circuit Breaker:** Jika API Bank mati total, *client* akan memutus sirkuit (*Fail-Fast*) agar server Anda tidak kehabisan RAM karena menunggu koneksi bodong.
- **Auto-Trace:** Otomatis membawa ID Pelacakan (OTEL/Trace ID) agar mudah ditelusuri dari *Dashboard*.

**Contoh Implementasi:**
```go
client := httpclient.New(httpclient.Options{
    Timeout:      5 * time.Second,
    MaxRetries:   3,
    EnableBreaker: true,
})
resp, err := client.Do(ctx, req) // Otomatis Retry jika gagal!
```

---

## 10. Pola Pesan Asinkron (*Messaging & Outbox Pattern*)
Selain integrasi Kafka standar (`messaging/`), `go-core` memfasilitasi pola **Outbox** (`messaging/outbox/`). 
Fitur ini mencegah hilangnya *event* ke Kafka saat *database* Anda berhasil di-*commit*, namun tiba-tiba listrik padam sebelum *event* sempat terkirim. *Outbox Worker* akan secara mandiri memastikannya terkirim (*At-Least-Once Delivery*).

---

## Ringkasan Alur Startup Standar di Service
Inilah cara setiap *microservice* Anda kelak dijalankan menggunakan komponen-komponen di atas:

```go
func main() {
    ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
    defer stop()

    // 1. Muat Config
    cfg, _ := coreconfig.Load(coreconfig.WithDotEnv(".env"))
    
    // 2. Siapkan Wadah Aplikasi (Otomatis init Logger, DB, Kafka, dll)
    application, _ := coreapp.New(ctx, cfg)
    
    // 3. Bangun Server Transport
    grpcServer, _ := coregrpc.New(application)
    gatewayServer, _ := coregateway.New(application, func(...) { /* Register Handler */ })
    
    // 4. Jalankan! (Otomatis mengurus Graceful Shutdown saat Ctrl+C ditekan)
    coreserver.Run(ctx, application, grpcServer, gatewayServer)
}
```
