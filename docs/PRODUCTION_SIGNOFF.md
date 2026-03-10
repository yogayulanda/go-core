# Production Sign-Off

This document defines release gates before production deploy.

## 1) Quality Gate (mandatory)

Run:

```bash
make quality-gate
```

Pass criteria:
- `go test ./...` passes
- `go test -race ./...` passes
- `go vet ./...` passes
- `golangci-lint run` passes
- `golangci-lint run -E gosec --tests=false` passes

## 2) Smoke Gate (mandatory, staging)

Run:

```bash
BASE_URL=https://staging.example.com make smoke-gate
```

Pass criteria:
- `/health` returns 200
- `/ready` returns 200
- `/version` returns 200

## 3) Performance Gate (mandatory, staging)

Use k6 scenario runner:

```bash
# steady load
BASE_URL=https://staging.example.com TARGET_PATH=/v1/your-endpoint make load-steady

# spike load
BASE_URL=https://staging.example.com TARGET_PATH=/v1/your-endpoint make load-spike

# soak test
BASE_URL=https://staging.example.com TARGET_PATH=/v1/your-endpoint make load-soak
```

Default thresholds:
- error rate < 1%
- p95 < 500ms
- p99 < 1000ms

Tune thresholds with env:
- `P95_MS`
- `P99_MS`
- `FAIL_RATE`

## 4) Failure Drill Gate (mandatory, staging)

Run with dependency stop/start commands (usually `kubectl` commands):

```bash
BASE_URL=https://staging.example.com \
STOP_DB_CMD="kubectl scale deploy/db --replicas=0 -n staging" \
START_DB_CMD="kubectl scale deploy/db --replicas=1 -n staging" \
STOP_KAFKA_CMD="kubectl scale sts/kafka --replicas=0 -n staging" \
START_KAFKA_CMD="kubectl scale sts/kafka --replicas=1 -n staging" \
make failure-drill
```

Pass criteria:
- baseline `/ready` = 200
- after dependency stop, `/ready` becomes 503 within timeout
- after recovery, `/ready` returns 200 within timeout

## 5) Evidence Gate (mandatory)

Create release evidence from template:

```bash
cp docs/RELEASE_EVIDENCE_TEMPLATE.md docs/RELEASE_EVIDENCE.md
```

Attach:
- command outputs
- dashboard screenshots (latency, errors, resource)
- failure drill timestamps and outcomes
- rollback readiness confirmation

Only deploy when all sections are marked `PASS`.
