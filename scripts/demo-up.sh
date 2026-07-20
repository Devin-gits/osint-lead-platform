#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
RUNTIME_DIR="${TMPDIR:-/tmp}/osint-lead-platform-demo"
API_LOG="${RUNTIME_DIR}/api.log"
UI_LOG="${RUNTIME_DIR}/ui.log"
API_PID="${RUNTIME_DIR}/api.pid"
UI_PID="${RUNTIME_DIR}/ui.pid"

port_in_use() {
  ss -ltn "( sport = :$1 )" 2>/dev/null | grep -q ":$1"
}

wait_for() {
  local url=$1
  for _ in {1..40}; do
    if curl -fsS "$url" >/dev/null 2>&1; then
      return 0
    fi
    sleep 0.25
  done
  return 1
}

if port_in_use 8080 || port_in_use 3000; then
  echo "Demo ports are already in use. Stop the existing services or run make demo-down." >&2
  exit 1
fi

mkdir -p "$RUNTIME_DIR"
rm -f "$API_PID" "$UI_PID"

(
  cd "$ROOT_DIR/services/control-plane"
  setsid env LISTEN_HOST=127.0.0.1 go run ./cmd/server >"$API_LOG" 2>&1 &
  echo $! >"$API_PID"
)

if ! wait_for "http://localhost:8080/healthz"; then
  echo "Control-plane did not become ready. See $API_LOG" >&2
  "$ROOT_DIR/scripts/demo-down.sh"
  exit 1
fi

(
  cd "$ROOT_DIR/ui/web-console"
  setsid env NEXT_PUBLIC_API_BASE_URL=http://localhost:8080 npm run dev >"$UI_LOG" 2>&1 &
  echo $! >"$UI_PID"
)

if ! wait_for "http://localhost:3000/leads"; then
  echo "Web console did not become ready. See $UI_LOG" >&2
  "$ROOT_DIR/scripts/demo-down.sh"
  exit 1
fi

echo "Demo ready:"
echo "  UI:  http://localhost:3000/leads"
echo "  API: http://localhost:8080"
echo "  API health: http://localhost:8080/healthz"
echo "Logs: $RUNTIME_DIR"
echo "Stop with: make demo-down"
