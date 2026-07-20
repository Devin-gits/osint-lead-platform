#!/usr/bin/env bash
set -euo pipefail

RUNTIME_DIR="${TMPDIR:-/tmp}/osint-lead-platform-demo"

for service in api ui; do
  pid_file="${RUNTIME_DIR}/${service}.pid"
  if [ -f "$pid_file" ]; then
    pid=$(cat "$pid_file")
    if kill -0 "$pid" 2>/dev/null; then
      kill -- "-${pid}"
      echo "Stopped ${service} process group (pid ${pid})."
    fi
    rm -f "$pid_file"
  fi
done
