# Go 1.26.9 Benchmark Evidence (2026-10-14)

## Method

For the runtime/dependency matrix, all variants ran interleaved for 20 rounds with `GOMAXPROCS=4`, `-run='^$'`, `-benchmem`, and `-count=1` per invocation:

- A: `main` with Go 1.24.13
- B: `main` with Go 1.26.9
- C: `chore/go-1.26` with Go 1.26.9

The GC-39 comparison used 20 interleaved before/after rounds with `GOMAXPROCS=4`, Go 1.26.9, and the same flags. Before is commit `ef1d749`; after is the GC-39 implementation before its code commit.

## Results

A/B/C medians and pairwise significance are in `matrix-benchstat.txt`. No significant gateway or unary interceptor latency change was found. Gateway allocations fell from 126 to 116 allocs/op between A and B due to the toolchain; B and C allocations were unchanged. `GOEXPERIMENT=nogreenteagc` is not supported by the available Go 1.26.9 toolchain (`unknown GOEXPERIMENT greenteagc`), so the alternate-GC variant was unavailable.

The GC-39 comparison is in `gc39-benchstat.txt`: the existing gateway benchmark measured 23.37 us/op before and 23.48 us/op after (+0.48%, p=0.461), with no significant latency regression. The new `GatewayRawHTTPEndToEnd` benchmark measured 20.73 us/op, 8.26 KiB/op, and 111 allocs/op.

The runs were measured on WSL, where variance limits practical regression detection to approximately 10%. The gateway benchmark uses `httptest.NewRecorder`; no TCP or grpc-gateway-to-gRPC proxy hop is included.

## Raw Samples

- `matrix-A-main-go1.24.13.txt`
- `matrix-B-main-go1.26.9.txt`
- `matrix-C-branch-go1.26.9.txt`
- `gc39-before-go1.26.9.txt`
- `gc39-after-go1.26.9.txt`
