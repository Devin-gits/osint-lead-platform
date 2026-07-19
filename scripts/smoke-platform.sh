#!/usr/bin/env bash
# Light platform smoke test.
#
# Runs scripts/smoke-extraction.sh first, then creates a lead with an email,
# runs email-validate, and checks the result is a structured status.
# Exits 0 on structured results (ok/unknown/skipped/error); non-zero on 5xx.

set -euo pipefail

BASE_URL="${SMOKE_BASE_URL:-http://localhost:8080}"

SMOKE_DIR="$(cd "$(dirname "$0")" && pwd)"

echo "==> Running extraction smoke (smoke-extraction.sh)"
"${SMOKE_DIR}/smoke-extraction.sh"

echo "==> Running email-validate smoke"

lead_resp=$(curl -s -w "\n%{http_code}" -X POST "${BASE_URL}/api/leads" \
  -H 'Content-Type: application/json' \
  -d '{"email":"support@github.com","permission_ref":"SMOKE-EMAIL-1"}')

http_code=$(echo "$lead_resp" | tail -n1)
body=$(echo "$lead_resp" | sed '$d')

if [ "$http_code" -ge 500 ]; then
  echo "ERROR: POST /api/leads returned ${http_code}" >&2
  echo "$body" >&2
  exit 1
fi

lead_id=$(echo "$body" | jq -r '.data.id // empty')
if [ -z "$lead_id" ]; then
  echo "ERROR: could not parse lead id" >&2
  echo "$body" >&2
  exit 1
fi

run_resp=$(curl -s -w "\n%{http_code}" -X POST "${BASE_URL}/api/leads/${lead_id}/run" \
  -H 'Content-Type: application/json' \
  -d '{"modules":["email-validate"]}')

http_code=$(echo "$run_resp" | tail -n1)
body=$(echo "$run_resp" | sed '$d')

if [ "$http_code" -ge 500 ]; then
  echo "ERROR: POST /api/leads/${lead_id}/run returned ${http_code}" >&2
  echo "$body" >&2
  exit 1
fi

email_validate=$(echo "$body" | jq '.data.email_validate')
echo "==> email_validate:"
echo "$email_validate" | jq '{status, deliverable, syntax_valid, has_mx_records, is_disposable, error}'

status=$(echo "$email_validate" | jq -r '.status // empty')
if [ -z "$status" ]; then
  echo "ERROR: email_validate status missing" >&2
  exit 1
fi

if [ "$status" != "ok" ] && [ "$status" != "unknown" ] && [ "$status" != "skipped" ] && [ "$status" != "error" ]; then
  echo "ERROR: unexpected email_validate status '${status}'" >&2
  exit 1
fi

echo "==> platform smoke passed"
