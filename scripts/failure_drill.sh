#!/usr/bin/env bash
set -euo pipefail

BASE_URL="${BASE_URL:-}"
if [[ -z "$BASE_URL" ]]; then
  echo "BASE_URL is required, example: BASE_URL=https://staging.example.com"
  exit 1
fi

READY_PATH="${READY_PATH:-/ready}"
HEALTH_PATH="${HEALTH_PATH:-/health}"
WAIT_TIMEOUT_SEC="${WAIT_TIMEOUT_SEC:-180}"
POLL_INTERVAL_SEC="${POLL_INTERVAL_SEC:-5}"
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

http_code() {
  local path="$1"
  curl "${curl_opts[@]}" "${BASE_URL}${path}"
}

wait_for_code() {
  local path="$1"
  local expected="$2"
  local label="$3"
  local waited=0

  while (( waited < WAIT_TIMEOUT_SEC )); do
    local code
    code="$(http_code "$path")"
    if [[ "$code" == "$expected" ]]; then
      echo "${label}: ${path} -> ${code}"
      return 0
    fi
    sleep "$POLL_INTERVAL_SEC"
    waited=$(( waited + POLL_INTERVAL_SEC ))
  done

  echo "${label}: timeout waiting ${path}=${expected}"
  return 1
}

run_cmd_if_set() {
  local cmd="$1"
  local label="$2"
  if [[ -z "$cmd" ]]; then
    echo "${label}: skipped (command not set)"
    return 0
  fi
  echo "${label}: ${cmd}"
  bash -lc "$cmd"
}

drill_required_dependency() {
  local name="$1"
  local stop_cmd="$2"
  local start_cmd="$3"

  if [[ -z "$stop_cmd" || -z "$start_cmd" ]]; then
    echo "[${name}] skipped (STOP/START command not fully provided)"
    return 0
  fi

  run_cmd_if_set "$stop_cmd" "[${name}] stop"
  wait_for_code "$READY_PATH" "503" "[${name}] ready down"

  run_cmd_if_set "$start_cmd" "[${name}] start"
  wait_for_code "$READY_PATH" "200" "[${name}] ready recovered"
}

drill_optional_dependency() {
  local name="$1"
  local stop_cmd="$2"
  local start_cmd="$3"

  if [[ -z "$stop_cmd" || -z "$start_cmd" ]]; then
    echo "[${name}] skipped (STOP/START command not fully provided)"
    return 0
  fi

  run_cmd_if_set "$stop_cmd" "[${name}] stop"
  wait_for_code "$HEALTH_PATH" "200" "[${name}] health still up"

  run_cmd_if_set "$start_cmd" "[${name}] start"
  wait_for_code "$HEALTH_PATH" "200" "[${name}] health recovered"
}

echo "failure drill baseline checks"
wait_for_code "$HEALTH_PATH" "200" "[baseline] health"
wait_for_code "$READY_PATH" "200" "[baseline] ready"

drill_required_dependency "db" "${STOP_DB_CMD:-}" "${START_DB_CMD:-}"
drill_required_dependency "kafka" "${STOP_KAFKA_CMD:-}" "${START_KAFKA_CMD:-}"
drill_optional_dependency "otlp" "${STOP_OTLP_CMD:-}" "${START_OTLP_CMD:-}"

echo "failure drill completed"
