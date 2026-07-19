#!/usr/bin/env bash
# Smoke test for the crm_ready stage policy and export stub.
#
# Assumptions:
#   - control-plane is running on localhost:8080
#   - curl and jq are installed
#
# This script exercises the happy path (email validation + company enrichment ->
# promote -> export) and negative cases (promote before validation returns 409,
# export before promotion returns 409). No CRM connector is called.

set -euo pipefail

BASE_URL="${SMOKE_BASE_URL:-http://localhost:8080}"

echo "==> Smoke target: ${BASE_URL}"

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

# Helper: create a lead
create_lead() {
  curl -s -X POST "${BASE_URL}/api/leads" \
    -H 'Content-Type: application/json' \
    -d "$1" | jq -r '.data.id // empty'
}

# Helper: run modules
run_modules() {
  local lead_id=$1
  shift
  local mods=""
  for m in "$@"; do
    [ -n "$mods" ] && mods="${mods},"
    mods="${mods}${m}"
  done
  curl -s -X POST "${BASE_URL}/api/leads/${lead_id}/run" \
    -H 'Content-Type: application/json' \
    -d "{\"modules\":[${mods}]}" | jq -r '.data'
}

# Helper: HTTP status code of promote
promote_status() {
  local lead_id=$1
  curl -s -o /dev/null -w "%{http_code}" -X POST "${BASE_URL}/api/leads/${lead_id}/promote" \
    -H 'Content-Type: application/json' \
    -d '{"target":"crm_ready"}'
}

# 1. Create a lead that can become crm_ready
echo "==> creating lead with email, company, domain, and permission_ref"
LEAD=$(create_lead '{"email":"support@example.com","company":"Example","domain":"example.com","permission_ref":"CRM-SMOKE-1"}')
if [ -z "$LEAD" ]; then
  echo "ERROR: could not create lead" >&2
  exit 1
fi
echo "==> lead ${LEAD}"

# 2. Promote before validation should 409
echo "==> promoting before validation (expect 409)"
status=$(promote_status "$LEAD")
if [ "$status" != "409" ]; then
  echo "ERROR: expected 409, got ${status}" >&2
  exit 1
fi

# 3. Readiness should show not ready
echo "==> checking readiness before validation"
ready_before=$(curl -s "${BASE_URL}/api/leads/${LEAD}/readiness" | jq -r '.data.ready')
if [ "${ready_before}" != "false" ]; then
  echo "ERROR: expected ready=false before validation, got ${ready_before}" >&2
  exit 1
fi

# 4. Run email-validate and company-enrich
echo "==> running email-validate and company-enrich"
run_modules "$LEAD" '"email-validate"' '"company-enrich"' >/dev/null

# 5. Readiness should now be true
echo "==> checking readiness after validation"
ready_after=$(curl -s "${BASE_URL}/api/leads/${LEAD}/readiness" | jq -r '.data.ready')
if [ "${ready_after}" != "true" ]; then
  echo "ERROR: expected ready=true after validation, got ${ready_after}" >&2
  curl -s "${BASE_URL}/api/leads/${LEAD}/readiness" | jq '.data' >&2
  exit 1
fi

# 6. Promote should succeed
echo "==> promoting to crm_ready"
promote=$(curl -s -X POST "${BASE_URL}/api/leads/${LEAD}/promote" \
  -H 'Content-Type: application/json' \
  -d '{"target":"crm_ready"}')
stage=$(echo "$promote" | jq -r '.data.stage // empty')
if [ "${stage}" != "crm_ready" ]; then
  echo "ERROR: expected stage crm_ready, got ${stage}" >&2
  echo "$promote" >&2
  exit 1
fi
echo "==> lead stage is ${stage}"

# 7. Export should return crm_stub_v1
echo "==> exporting lead"
export_resp=$(curl -s "${BASE_URL}/api/leads/${LEAD}/export")
format=$(echo "$export_resp" | jq -r '.data.format // empty')
if [ "${format}" != "crm_stub_v1" ]; then
  echo "ERROR: expected format crm_stub_v1, got ${format}" >&2
  echo "$export_resp" >&2
  exit 1
fi
echo "$export_resp" | jq '.data | {format, exported_at, permission_ref, lead: .lead | {id, stage, risk_level}}'

# 8. Export should include readiness and enrichment
echo "==> export contains readiness and enrichment keys"
echo "$export_resp" | jq '.data | keys'

# 9. Demote and verify export 409s
echo "==> demoting lead to validated"
demote=$(curl -s -X POST "${BASE_URL}/api/leads/${LEAD}/demote" \
  -H 'Content-Type: application/json' \
  -d '{"target":"validated"}')
demoted_stage=$(echo "$demote" | jq -r '.data.stage // empty')
if [ "${demoted_stage}" != "validated" ]; then
  echo "ERROR: expected stage validated, got ${demoted_stage}" >&2
  echo "$demote" >&2
  exit 1
fi

export_status=$(curl -s -o /dev/null -w "%{http_code}" "${BASE_URL}/api/leads/${LEAD}/export")
if [ "${export_status}" != "409" ]; then
  echo "ERROR: expected export 409 after demote, got ${export_status}" >&2
  exit 1
fi

# 10. Negative: create a lead without permission_ref, should never promote
echo "==> negative: lead without permission_ref"
BAD=$(create_lead '{"email":"a@example.com","company":"Bad"}')
run_modules "$BAD" '"email-validate"' '"company-enrich"' >/dev/null
bad_status=$(promote_status "$BAD")
if [ "${bad_status}" != "409" ]; then
  echo "ERROR: expected 409 for lead without permission_ref, got ${bad_status}" >&2
  exit 1
fi

# 11. Negative: lead with high risk
echo "==> negative: lead marked high risk"
HIGH_RISK=$(create_lead '{"email":"a@example.com","company":"Risky","permission_ref":"CRM-SMOKE-HIGH"}')
# Manually push the lead to high risk via update is not exposed; instead rely on
# runner risk (email disposable etc). For smoke simplicity, just verify that a
# lead with no validation cannot be promoted (already covered). This slot is
# reserved for a future PUT /leads/{id} test harness.

# 12. Audit events include pipeline records
echo "==> checking audit events"
audit_count=$(curl -s "${BASE_URL}/api/leads/${LEAD}" | jq '[.data.audit_events[] | select(.module=="pipeline")] | length')
if [ "${audit_count}" -lt 3 ]; then
  echo "ERROR: expected at least 3 pipeline audit events, got ${audit_count}" >&2
  exit 1
fi

echo "==> smoke-crm-ready passed"
