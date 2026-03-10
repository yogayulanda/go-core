#!/usr/bin/env bash
set -euo pipefail

SCENARIO="${1:-}"
if [[ -z "$SCENARIO" ]]; then
  echo "usage: scripts/load_gate.sh <steady|spike|soak>"
  exit 1
fi

case "$SCENARIO" in
  steady|spike|soak) ;;
  *)
    echo "invalid scenario: $SCENARIO"
    exit 1
    ;;
esac

if ! command -v k6 >/dev/null 2>&1; then
  echo "k6 not found. install: https://k6.io/docs/get-started/installation/"
  exit 1
fi

BASE_URL="${BASE_URL:-}"
if [[ -z "$BASE_URL" ]]; then
  echo "BASE_URL is required, example: BASE_URL=https://staging.example.com"
  exit 1
fi

TARGET_PATH="${TARGET_PATH:-/health}"
P95_MS="${P95_MS:-500}"
P99_MS="${P99_MS:-1000}"
FAIL_RATE="${FAIL_RATE:-0.01}"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCRIPT_FILE="${SCRIPT_DIR}/k6/http_gateway.js"

echo "running load scenario=${SCENARIO} base_url=${BASE_URL} target_path=${TARGET_PATH}"
k6 run \
  -e SCENARIO="$SCENARIO" \
  -e BASE_URL="$BASE_URL" \
  -e TARGET_PATH="$TARGET_PATH" \
  -e P95_MS="$P95_MS" \
  -e P99_MS="$P99_MS" \
  -e FAIL_RATE="$FAIL_RATE" \
  "$SCRIPT_FILE"
