#!/usr/bin/env bash
# Smoke test for async module worker queue.
#
# Assumptions:
#   - control-plane is running on localhost:8080
#   - curl and jq are installed

set -euo pipefail

BASE_URL="${SMOKE_BASE_URL:-http://localhost:8080}"

echo "==> Smoke target: ${BASE_URL}"

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

LEAD=$(curl -s -X POST "${BASE_URL}/api/leads" \
  -H 'Content-Type: application/json' \
  -d '{"email":"support@example.com","company":"Example","domain":"example.com","permission_ref":"ASYNC-1"}' | jq -r '.data.id // empty')
if [ -z "$LEAD" ]; then
  echo "ERROR: could not create lead" >&2
  exit 1
fi
echo "==> lead ${LEAD}"

echo "==> enqueuing email-validate (expect 202 + run_id)"
run_resp=$(curl -s -X POST "${BASE_URL}/api/leads/${LEAD}/run" \
  -H 'Content-Type: application/json' \
  -d '{"modules":["email-validate"]}')
run_id=$(echo "$run_resp" | jq -r '.data.run_id // empty')
status=$(echo "$run_resp" | jq -r '.data.status // empty')
if [ -z "$run_id" ] || [ "$status" != "queued" ]; then
  echo "ERROR: expected 202 queued response, got ${run_resp}" >&2
  exit 1
fi
echo "==> run ${run_id} queued"

echo "==> polling run status"
for i in {1..60}; do
  status=$(curl -s "${BASE_URL}/api/runs/${run_id}" | jq -r '.data.status // empty')
  echo "run status: ${status}"
  if [ "$status" = "completed" ] || [ "$status" = "partial" ] || [ "$status" = "failed" ]; then
    break
  fi
  sleep 1
done

if [ "$status" != "completed" ]; then
  echo "ERROR: expected completed, got ${status}" >&2
  exit 1
fi

echo "==> lead has results and recomputed risk"
lead=$(curl -s "${BASE_URL}/api/leads/${LEAD}")
echo "$lead" | jq '.data | {stage, risk_level, risk_score, email_validate: .email_validate | {status}}'
email_status=$(echo "$lead" | jq -r '.data.email_validate.status // empty')
risk_level=$(echo "$lead" | jq -r '.data.risk_level // empty')
if [ "$email_status" != "ok" ]; then
  echo "ERROR: expected email_validate ok, got ${email_status}" >&2
  exit 1
fi
if [ "$risk_level" = "unknown" ]; then
  echo "ERROR: expected risk_level recomputed, got unknown" >&2
  exit 1
fi

echo "==> smoke-async passed"
