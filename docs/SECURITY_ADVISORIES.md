# Security Advisories

## Scope

Assessment refreshed with `govulncheck` on 2026-10-14 using `GOTOOLCHAIN=go1.24.13`. The scanner binary was built with Go 1.26.9, but the analyzed module/toolchain target was Go 1.24.13. The repository directive remains `go 1.24.3`.

`called` means govulncheck reported a source call trace into the affected package/symbol. `imported only` means the module or standard-library package was reported without such a trace. A call trace is reachability evidence, not proof of exploitability; absence of a trace is not proof that a package cannot be reached dynamically.

## Remediated

| Advisory | Package | Selected version | Required Go | Status |
|---|---|---:|---:|---|
| GO-2026-4762 | `google.golang.org/grpc` | v1.79.3 | 1.24.0 | fixed; scanner no longer reports it |
| GO-2026-5506 | `go.opentelemetry.io/otel`, `/trace`, `/metric` | v1.41.0 | 1.24.0 | fixed; scanner no longer reports it |
| GO-2026-5841 | `github.com/klauspost/compress` | v1.18.7 | 1.24.0 | fixed; scanner no longer reports it |

All selected fixes are stable releases older than two weeks and are compatible with Go 1.24.

## Third-Party Modules

Fix versions below are the lowest stable versions identified by the advisory database for the affected packages. Fixes requiring Go 1.25 or newer are deferred under the current Go 1.24.13 image constraint.

| Advisory | Affected package | Current version | Fixed version | Required Go | Status |
|---|---|---:|---:|---:|---|
| GO-2026-4918 | `golang.org/x/net/http2` | x/net v0.49.0 | x/net v0.53.0 | 1.25.0 | imported only |
| GO-2026-5025, GO-2026-5027, GO-2026-5028, GO-2026-5029, GO-2026-5030 | `golang.org/x/net/html` | x/net v0.49.0 | x/net v0.55.0 | 1.25.0 | imported only |
| GO-2026-5026 | `golang.org/x/net/idna` | x/net v0.49.0 | x/net v0.55.0 | 1.25.0 | imported only |
| GO-2026-5942 | `golang.org/x/net/dns/dnsmessage` | x/net v0.49.0 | x/net v0.56.0 | 1.25.0 | imported only |
| GO-2026-6603, GO-2026-6611, GO-2026-6612, GO-2026-6617 | `golang.org/x/net/http2` | x/net v0.49.0 | x/net v0.60.0 | 1.26.0 | called |
| GO-2026-6610 | `golang.org/x/net/http2` | x/net v0.49.0 | x/net v0.60.0 | 1.26.0 | imported only |
| GO-2026-5005, GO-2026-5006, GO-2026-5033 | `golang.org/x/crypto/ssh/agent` | x/crypto v0.47.0 | x/crypto v0.52.0 | 1.25.0 | imported only |
| GO-2026-5013, GO-2026-5014, GO-2026-5015, GO-2026-5016, GO-2026-5017, GO-2026-5018, GO-2026-5019, GO-2026-5020, GO-2026-5023 | `golang.org/x/crypto/ssh` | x/crypto v0.47.0 | x/crypto v0.52.0 | 1.25.0 | imported only |
| GO-2026-5021 | `golang.org/x/crypto/ssh/knownhosts` | x/crypto v0.47.0 | x/crypto v0.52.0 | 1.25.0 | imported only |
| GO-2026-6303 | `golang.org/x/crypto/ssh` | x/crypto v0.47.0 | x/crypto v0.55.0 | 1.25.0 | imported only |
| GO-2026-6354, GO-2026-6355 | `golang.org/x/crypto/ssh` | x/crypto v0.47.0 | x/crypto v0.56.0 | 1.26.0 | imported only |
| GO-2026-5932 | `golang.org/x/crypto/openpgp` and subpackages | x/crypto v0.47.0 | no fix available | n/a | imported only |
| GO-2026-5024 | `golang.org/x/sys/windows` | x/sys v0.40.0 | x/sys v0.44.0 | 1.25.0 | imported only |
| GO-2026-5970 | `golang.org/x/text/unicode/norm` | x/text v0.34.0 | x/text v0.39.0 | 1.25.0 | called |
| GO-2026-6629 | `golang.org/x/text/secure/precis` | x/text v0.34.0 | x/text v0.41.0 | 1.25.0 | imported only |
| GO-2026-6061 | `google.golang.org/grpc/internal/transport`, `internal/xds/rbac` | grpc v1.79.3 | grpc v1.82.1 | 1.25.0 | called |
| GO-2026-6443 | `google.golang.org/grpc/internal/transport`, `internal/xds/server` | grpc v1.79.3 | grpc v1.82.2 | 1.25.0 | called |
| GO-2026-6348 | `google.golang.org/grpc/internal/transport`, `grpc/mem` | grpc v1.79.3 | grpc v1.83.1 | 1.25.0 | called |
| GO-2026-6441 | `google.golang.org/grpc/internal/xds/httpfilter/rbac` | grpc v1.79.3 | grpc v1.83.1 | 1.25.0 | imported only |
| GO-2026-5158 | `go.opentelemetry.io/otel/baggage`, `/propagation` | otel v1.41.0 | otel v1.42.0 | 1.25.0 | called |
| GO-2026-5426 | `go.opentelemetry.io/otel/sdk/resource` | sdk v1.40.0 | sdk v1.43.0 | 1.25.0 | called |
| GO-2026-6505 | OTLP trace exporter modules and `go.opentelemetry.io/otel/sdk` | v1.40.0 | v1.45.0 | 1.25.0 | called |

GO-2026-5932 concerns the deprecated, unmaintained OpenPGP implementation and has no upstream fix version. It remains listed even though govulncheck found no source call trace.

## Go Standard Library

These findings are evaluated against Go 1.24.13. The “fixed Go” column is the lowest stable Go toolchain version listed as fixed by the advisory database; it is not a dependency-module minimum. All fixes are newer than the current Go 1.24.13 image. Rows with multiple IDs share the same package, fix release, and reachability status.

| Advisory | Affected package | Fixed Go | Status |
|---|---|---:|---|
| GO-2026-4601 | `net/url` | 1.25.8 | called |
| GO-2026-4602 | `os` | 1.25.8 | called |
| GO-2026-4603 | `html/template` | 1.25.8 | called |
| GO-2026-4864 | `internal/syscall/unix` | 1.25.9 | imported only |
| GO-2026-4865 | `html/template` | 1.25.9 | called |
| GO-2026-4869 | `archive/tar` | 1.25.9 | imported only |
| GO-2026-4870 | `crypto/tls` | 1.25.9 | called |
| GO-2026-4946, GO-2026-4947 | `crypto/x509` | 1.25.9 | called |
| GO-2026-4970 | `os` | 1.25.12 | imported only |
| GO-2026-4971 | `net` | 1.25.10 | called |
| GO-2026-4976 | `net/http/httputil` | 1.25.10 | imported only |
| GO-2026-4977, GO-2026-4986 | `net/mail` | 1.25.10 | imported only |
| GO-2026-4980, GO-2026-4982 | `html/template` | 1.25.10 | called |
| GO-2026-4981 | `net` | 1.25.10 | imported only |
| GO-2026-5026 | `net/http`, `net/http/internal/http2` | 1.25.13 | called |
| GO-2026-5037 | `crypto/x509` | 1.25.11 | called |
| GO-2026-5038 | `mime` | 1.25.11 | imported only |
| GO-2026-5039 | `net/textproto` | 1.25.11 | called |
| GO-2026-5856 | `crypto/tls` | 1.25.12 | called |
| GO-2026-5972 | `encoding/asn1` | 1.25.13 | called |
| GO-2026-6088 | `encoding/xml` | 1.25.13 | called |
| GO-2026-6089 | `net/http` | 1.25.13 | called |
| GO-2026-6090 | `crypto/tls` | 1.25.13 | called |
| GO-2026-6091 | `html/template` | 1.25.13 | called |
| GO-2026-6218 | `net/url` | 1.25.13 | called |
| GO-2026-4918 | `net/http` (stdlib) | 1.25.10 | called |
| GO-2026-5942 | `net` | 1.26.6 | imported only |
| GO-2026-6599, GO-2026-6600 | `html/template` | 1.26.9 | called |
| GO-2026-6603 | `net/http`, `net/http/internal/http2` | 1.26.9 | called |
| GO-2026-6604 | `internal/syscall/windows`, `os` | 1.26.9 | imported only |
| GO-2026-6605 | `net/http`, `net/http/httputil` | 1.26.9 | called |
| GO-2026-6607 | `crypto/tls` | 1.26.9 | called |
| GO-2026-6608 | `mime/multipart`, `net/textproto` | 1.26.9 | called |
| GO-2026-6609 | `net/http` | 1.26.9 | imported only |
| GO-2026-6610 | `net/http`, `net/http/internal/http2` | 1.26.9 | called |
| GO-2026-6611, GO-2026-6612 | `net/http`, `net/http/internal/http2` | 1.26.9 | called |
| GO-2026-6613 | `net/http` | 1.26.9 | called |
| GO-2026-6617 | `net/http`, `net/http/internal/http2` | 1.26.9 | called |

## Image and Upgrade Plan

- Keep the source directive at `go 1.24.3`.
- Use Go 1.24.13 for the current build image; it is the final patch release for the Go 1.24 line.
- Plan the next toolchain/image upgrade to Go 1.26.x, not Go 1.27. Re-evaluate the deferred module upgrades against that image before upgrading dependencies.
- No major dependency upgrades or fixes requiring Go 1.25+ were included in this remediation.
