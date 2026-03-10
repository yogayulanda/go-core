#!/usr/bin/env bash
set -euo pipefail

BASE_URL="${BASE_URL:-}"
if [[ -z "$BASE_URL" ]]; then
  echo "BASE_URL is required, example: BASE_URL=https://staging.example.com"
  exit 1
fi

HEALTH_PATH="${HEALTH_PATH:-/health}"
READY_PATH="${READY_PATH:-/ready}"
VERSION_PATH="${VERSION_PATH:-/version}"
REQUEST_TIMEOUT_SEC="${REQUEST_TIMEOUT_SEC:-10}"

curl_opts=(
  -sS
  -o /dev/null
  -w "%{http_code}"
  --max-time "$REQUEST_TIMEOUT_SEC"
)

if [[ "${CURL_INSECURE:-0}" == "1" ]]; then
  curl_opts+=(-k)
fi

check_status() {
  local path="$1"
  local expected="$2"
  local code
  code="$(curl "${curl_opts[@]}" "${BASE_URL}${path}")"
  if [[ "$code" != "$expected" ]]; then
    echo "smoke failed: ${path} expected ${expected}, got ${code}"
    exit 1
  fi
  echo "smoke ok: ${path} -> ${code}"
}

check_status "$HEALTH_PATH" "200"
check_status "$READY_PATH" "200"
check_status "$VERSION_PATH" "200"

echo "smoke gate passed"
