#!/usr/bin/env bash
# Smoke test for deterministic risk_score v2.
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

# 1. Lead with no contact/domain signals: risk unknown.
NO_SIGS=$(curl -s -X POST "${BASE_URL}/api/leads" \
  -H 'Content-Type: application/json' \
  -d '{"company":"Stealth"}' | jq -r '.data.id // empty')
if [ -z "$NO_SIGS" ]; then
  echo "ERROR: could not create no-signal lead" >&2
  exit 1
fi
echo "==> no-signal lead ${NO_SIGS}"
level=$(curl -s "${BASE_URL}/api/leads/${NO_SIGS}/risk" | jq -r '.data.level')
if [ "$level" != "unknown" ]; then
  echo "ERROR: expected level unknown for lead with no signals, got ${level}" >&2
  exit 1
fi
echo "==> no-signal risk level: ${level}"

# 2. Lead with email but no validation: low with a positive unvalidated-contact score.
LEAD=$(curl -s -X POST "${BASE_URL}/api/leads" \
  -H 'Content-Type: application/json' \
  -d '{"email":"support@example.com","company":"Example","domain":"example.com","permission_ref":"RISK-SMOKE-1"}' | jq -r '.data.id // empty')
if [ -z "$LEAD" ]; then
  echo "ERROR: could not create lead" >&2
  exit 1
fi
echo "==> lead ${LEAD}"

echo "==> risk before validation (expect low with unvalidated-contact score)"
before=$(curl -s "${BASE_URL}/api/leads/${LEAD}/risk" | jq -r '[.data.level, .data.score // "null"] | @tsv')
echo "before: $before"
before_level=$(echo "$before" | awk '{print $1}')
if [ "$before_level" != "low" ]; then
  echo "ERROR: expected level low, got $before_level" >&2
  exit 1
fi

echo "==> running email-validate and company-enrich"
curl -s -X POST "${BASE_URL}/api/leads/${LEAD}/run" \
  -H 'Content-Type: application/json' \
  -d '{"modules":["email-validate","company-enrich"]}' > /dev/null

echo "==> risk after validation (expect low score 0)"
after=$(curl -s "${BASE_URL}/api/leads/${LEAD}/risk" | jq -r '[.data.level, .data.score // "null"] | @tsv')
echo "after: $after"
after_level=$(echo "$after" | awk '{print $1}')
after_score=$(echo "$after" | awk '{print $2}')
if [ "$after_level" != "low" ]; then
  echo "ERROR: expected level low, got $after_level" >&2
  exit 1
fi
if [ "$after_score" = "null" ] || [ -z "$after_score" ]; then
  echo "ERROR: expected numeric score, got $after_score" >&2
  exit 1
fi

echo "==> factors present"
curl -s "${BASE_URL}/api/leads/${LEAD}/risk" | jq '.data.factors | length'

echo "==> smoke-risk passed"
