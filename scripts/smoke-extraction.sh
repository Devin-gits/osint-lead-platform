#!/usr/bin/env bash
# Smoke test for the control-plane extraction module.
#
# Assumptions:
#   - control-plane is running on localhost:8080
#   - curl and jq are installed
#
# This script exits 0 when the extraction pipeline returns a structured result
# (ok, partial, skipped, or error). It exits non-zero on HTTP 5xx, missing
# extraction key, or panic output.
#
# Set SMOKE_REQUIRE_OK=1 to require status "ok" or "partial" (useful in a fully
# provisioned demo environment).

set -euo pipefail

BASE_URL="${SMOKE_BASE_URL:-http://localhost:8080}"
REQUIRE_OK="${SMOKE_REQUIRE_OK:-0}"

echo "==> Smoke target: ${BASE_URL}"
echo "==> SMOKE_REQUIRE_OK=${REQUIRE_OK}"

# Health check / wait briefly for the server
try_count=0
while ! curl -s "${BASE_URL}/api/modules" > /dev/null 2>&1; do
  try_count=$((try_count + 1))
  if [ "$try_count" -ge 10 ]; then
    echo "ERROR: control-plane not reachable at ${BASE_URL}/api/modules" >&2
    exit 1
  fi
  echo "Waiting for control-plane... (${try_count}/10)"
  sleep 1
done

# Verify extraction is registered as available
extraction_status=$(curl -s "${BASE_URL}/api/modules" | jq -r '.data[] | select(.name=="extraction") | .dev_status')
if [ "${extraction_status}" != "available" ]; then
  echo "ERROR: extraction module is not 'available' (got: ${extraction_status})" >&2
  exit 1
fi
echo "==> extraction module is available"

# Create a lead with URL and permission_ref
lead_resp=$(curl -s -w "\n%{http_code}" -X POST "${BASE_URL}/api/leads" \
  -H 'Content-Type: application/json' \
  -d '{"url":"https://example.com","permission_ref":"SMOKE-1"}')

http_code=$(echo "$lead_resp" | tail -n1)
body=$(echo "$lead_resp" | sed '$d')

if [ "$http_code" -ge 500 ]; then
  echo "ERROR: POST /api/leads returned ${http_code}" >&2
  echo "$body" >&2
  exit 1
fi
if [ "$http_code" -ge 400 ]; then
  echo "WARNING: POST /api/leads returned ${http_code}: ${body}" >&2
fi

lead_id=$(echo "$body" | jq -r '.data.id // empty')
if [ -z "$lead_id" ]; then
  echo "ERROR: could not parse lead id from response" >&2
  echo "$body" >&2
  exit 1
fi
echo "==> created lead ${lead_id}"

wait_for_run() {
  local run_id=$1
  local status
  for i in {1..60}; do
    status=$(curl -s "${BASE_URL}/api/runs/${run_id}" | jq -r '.data.status // empty')
    if [ "$status" = "completed" ] || [ "$status" = "partial" ] || [ "$status" = "failed" ]; then
      printf '%s\n' "$status"
      return 0
    fi
    sleep 1
  done
  echo "ERROR: run ${run_id} did not reach a terminal status" >&2
  return 1
}

# Run extraction
run_resp=$(curl -s -w "\n%{http_code}" -X POST "${BASE_URL}/api/leads/${lead_id}/run" \
  -H 'Content-Type: application/json' \
  -d '{"modules":["extraction"]}')

http_code=$(echo "$run_resp" | tail -n1)
body=$(echo "$run_resp" | sed '$d')

if [ "$http_code" != "202" ]; then
  echo "ERROR: POST /api/leads/${lead_id}/run returned ${http_code}" >&2
  echo "$body" >&2
  exit 1
fi

run_id=$(echo "$body" | jq -r '.data.run_id // empty')
if [ -z "$run_id" ]; then
  echo "ERROR: run response did not include run_id" >&2
  echo "$body" >&2
  exit 1
fi
wait_for_run "$run_id" >/dev/null

lead_get=$(curl -s "${BASE_URL}/api/leads/${lead_id}")
extraction=$(echo "$lead_get" | jq '.data.extraction')
if [ "$extraction" = "null" ] || [ -z "$extraction" ]; then
  echo "ERROR: extraction result missing after run completion" >&2
  echo "$lead_get" >&2
  exit 1
fi

status=$(echo "$extraction" | jq -r '.status')
echo "==> extraction status: ${status}"
echo "$extraction" | jq '{status, source_tool, confidence, fields, error, metadata}'

# Get the audit event
lead_get=$(curl -s "${BASE_URL}/api/leads/${lead_id}")
audit=$(echo "$lead_get" | jq '.data.audit_events[0]')
echo "==> audit event:"
echo "$audit" | jq '{module, status, legal_basis, subject, raw_stderr_json: .raw_stderr_json}'

# Validate expected structured statuses
if [ "$status" != "ok" ] && [ "$status" != "partial" ] && [ "$status" != "skipped" ] && [ "$status" != "error" ]; then
  echo "ERROR: unexpected extraction status '${status}'" >&2
  exit 1
fi

if [ "$REQUIRE_OK" = "1" ] && [ "$status" != "ok" ] && [ "$status" != "partial" ]; then
  echo "ERROR: SMOKE_REQUIRE_OK=1 but status was '${status}'" >&2
  exit 1
fi

# Optional: missing permission_ref path should be skipped, not an error.
# This block always exits 0 for skipped and is not gated by SMOKE_REQUIRE_OK.
no_perm_resp=$(curl -s -X POST "${BASE_URL}/api/leads" \
  -H 'Content-Type: application/json' \
  -d '{"url":"https://example.com"}')
no_perm_id=$(echo "$no_perm_resp" | jq -r '.data.id // empty')
if [ -n "$no_perm_id" ]; then
  no_perm_run=$(curl -s -X POST "${BASE_URL}/api/leads/${no_perm_id}/run" \
    -H 'Content-Type: application/json' \
    -d '{"modules":["extraction"]}')
  no_perm_run_id=$(echo "$no_perm_run" | jq -r '.data.run_id // empty')
  if [ -z "$no_perm_run_id" ]; then
    echo "ERROR: missing permission_ref run did not include run_id" >&2
    echo "$no_perm_run" >&2
    exit 1
  fi
  wait_for_run "$no_perm_run_id" >/dev/null
  no_perm_lead=$(curl -s "${BASE_URL}/api/leads/${no_perm_id}")
  no_perm_status=$(echo "$no_perm_lead" | jq -r '.data.extraction.status // empty')
  no_perm_reason=$(echo "$no_perm_lead" | jq -r '.data.extraction.reason // empty')
  echo "==> missing permission_ref lead: extraction status=${no_perm_status}, reason=${no_perm_reason}"
  if [ "${no_perm_status}" != "skipped" ]; then
    echo "ERROR: expected missing permission_ref extraction to be skipped, got '${no_perm_status}'" >&2
    exit 1
  fi
fi

echo "==> smoke passed"
