# Security Advisories

## Scan Results

Scanned 2026-10-14 with govulncheck v1.9.0 and Go 1.26.9. The vulnerability database was last updated 2026-10-08. Pre-upgrade scan on Go 1.24.13 reported 39 called vulnerabilities across five modules and the standard library, plus 7 imported-only package findings and 29 module-only findings. After the toolchain and dependency upgrades, the scan reports **0 called vulnerabilities**, **0 vulnerable imported packages**, and **1 vulnerable required module without a call trace**.

`called` means govulncheck found a source call trace to the affected symbol. `imported only` means a vulnerable package/module is in the dependency graph without a reported call trace. These are reachability signals, not proof of exploitability or proof that dynamic reachability is impossible.

## Remaining Finding

| Advisory | Module/package | Selected version | Status |
|---|---|---:|---|
| GO-2026-5932 | `golang.org/x/crypto/openpgp` | x/crypto v0.57.0 | Module-only; no upstream fix. The deprecated OpenPGP package is unmaintained and unsafe by design; no call trace was found. |

## Dependency Upgrades

Selected advisory fixes and their required graph versions are listed below. Release ages are measured at 2026-10-14. The x/net v0.60.0 security minimum is a documented exception to the usual 14-day age preference, approved because it is an official golang.org/x fix and necessary to resolve called advisories.

| Module | From | To | Release date | Age (days) | Reason |
|---|---:|---:|---:|---:|---|
| `golang.org/x/net` | v0.49.0 | v0.60.0 | 2026-10-08 | 6 | Advisory fix minimum; approved <14-day security exception. |
| `golang.org/x/crypto` | v0.47.0 | v0.57.0 | 2026-09-08 | 36 | Advisory minimum was v0.56.0; v0.57.0 is required by x/net v0.60.0. |
| `golang.org/x/sys` | v0.40.0 | v0.48.0 | 2026-08-31 | 44 | Advisory minimum was v0.44.0; v0.48.0 is required by x/net/x/crypto. |
| `golang.org/x/text` | v0.34.0 | v0.42.0 | 2026-09-08 | 36 | Advisory minimum was v0.41.0; v0.42.0 is required by x/net/x/crypto. |
| `google.golang.org/grpc` | v1.79.3 | v1.83.2 | 2026-08-25 | 50 | Advisory fixes; v1.83.2 also fixes GO-2026-6443 found by the first post-upgrade scan. |
| `go.opentelemetry.io/otel` | v1.41.0 | v1.45.0 | 2026-08-03 | 72 | Advisory fix minimum. |
| `go.opentelemetry.io/otel/trace` | v1.41.0 | v1.45.0 | 2026-08-03 | 72 | Aligned with OTel release. |
| `go.opentelemetry.io/otel/metric` | v1.41.0 | v1.45.0 | 2026-08-03 | 72 | Aligned with OTel release. |
| `go.opentelemetry.io/otel/sdk` | v1.40.0 | v1.45.0 | 2026-08-03 | 72 | Advisory fix minimum. |
| `go.opentelemetry.io/otel/exporters/otlp/otlptrace` | v1.40.0 | v1.45.0 | 2026-08-03 | 72 | Aligned with OTel SDK/exporter fix. |
| `go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc` | v1.40.0 | v1.45.0 | 2026-08-03 | 72 | Advisory fix minimum. |

Additional transitive upgrades:

| Module | From | To | Release date | Age (days) | Why |
|---|---:|---:|---:|---:|---|
| `github.com/grpc-ecosystem/grpc-gateway/v2` | v2.27.8 | v2.29.0 | 2026-04-15 | 182 | Required by grpc dependency graph. |
| `github.com/go-logr/logr` | v1.4.3 | v1.4.4 | 2026-07-20 | 86 | Required by OTel dependency graph. |
| `go.opentelemetry.io/proto/otlp` | v1.9.0 | v1.11.0 | 2026-07-22 | 84 | Required by OTel exporter. |
| `golang.org/x/sync` | v0.19.0 | v0.23.0 | 2026-08-31 | 44 | Required by the selected security dependency graph. |
| `google.golang.org/genproto/googleapis/api` | 2026-02-09 revision | 2026-08-03 revision | 2026-08-03 | 72 | Required by grpc/gateway graph. |
| `google.golang.org/genproto/googleapis/rpc` | 2026-02-09 revision | 2026-08-03 revision | 2026-08-03 | 72 | Required by grpc graph. |

## Go Standard Library

Go 1.26.9 resolves the standard-library findings recorded by the previous Go 1.24.13 scan, including `net/url`, `net/http`, `crypto/tls`, `crypto/x509`, `os`, `html/template`, `mime/multipart`, `net/textproto`, and `encoding/asn1/xml`. No standard-library advisory remains in the current govulncheck result.

## Rolling N-1 Policy

- Use the latest patch release in the N-1 Go minor line. Advance the minor target with Go's February/August releases.
- Apply Go security patches no later than 14 days after release.
- Set the `go` directive to the minimum required by the selected dependency graph, as determined by `go mod tidy`; do not copy the installed toolchain patch into the directive.
- Prefer stable dependency versions older than 14 days. An official maintainer's minimum security fix may be used sooner when required to remediate an advisory; document the exception, release date, and age.
- Re-run govulncheck after dependency or toolchain changes and retain imported-only/no-fix findings explicitly.
