#!/usr/bin/env bash
# Operator smoke test for the control-plane API.
#
# Assumptions:
#   - control-plane is running on localhost:8080
#   - curl and jq are installed
#
# This script exercises the company-enrich module end-to-end and verifies
# core module registry / lead lifecycle behaviour. It exits 0 on success.
#
# Set SMOKE_BASE_URL to override the API base URL.

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

# Verify all six modules are registered as available
for module in company-enrich domain-intel email-validate extraction phone-validate social-footprint; do
  status=$(curl -s "${BASE_URL}/api/modules" | jq -r ".data[] | select(.name==\"${module}\") | .dev_status")
  if [ "${status}" != "available" ]; then
    echo "ERROR: ${module} module is not 'available' (got: ${status})" >&2
    exit 1
  fi
  echo "==> ${module} module is available"
done

# Helper: create a lead and return its id
create_lead() {
  curl -s -X POST "${BASE_URL}/api/leads" \
    -H 'Content-Type: application/json' \
    -d "$1" | jq -r '.data.id // empty'
}

# Helper: run a module and return the namespaced result key
run_module() {
  local lead_id=$1
  local module=$2
  local key=${3:-${module//-/_}}
  curl -s -X POST "${BASE_URL}/api/leads/${lead_id}/run" \
    -H 'Content-Type: application/json' \
    -d "{\"modules\":[\"${module}\"]}" | jq -r ".data.${key} // empty"
}

# 1. Company enrich: domain + company + permission_ref => ok
echo "==> creating lead with domain + company + permission_ref"
lead_id=$(create_lead '{"domain":"example.com","company":"Example","permission_ref":"SMOKE-1"}')
if [ -z "$lead_id" ]; then
  echo "ERROR: could not create lead" >&2
  exit 1
fi
echo "==> created lead ${lead_id}"

company=$(run_module "$lead_id" "company-enrich" "company_enrich")
status=$(echo "$company" | jq -r '.status // empty')
echo "==> company-enrich status: ${status}"
if [ "${status}" != "ok" ]; then
  echo "ERROR: expected company-enrich status 'ok' for domain+company lead, got '${status}'" >&2
  echo "$company" >&2
  exit 1
fi
name=$(echo "$company" | jq -r '.fields.name // empty')
if [ "${name}" != "Example" ]; then
  echo "ERROR: expected company name 'Example', got '${name}'" >&2
  echo "$company" >&2
  exit 1
fi
echo "$company" | jq '{status, source_tool, confidence, fields: .fields | {domain, name, website}, metadata: .metadata | {legal_basis, permission_ref}}'

# 2. Company enrich: domain-only => partial with empty name
echo "==> creating domain-only lead"
lead2_id=$(create_lead '{"domain":"example.com","permission_ref":"SMOKE-2"}')
if [ -z "$lead2_id" ]; then
  echo "ERROR: could not create domain-only lead" >&2
  exit 1
fi
echo "==> created lead ${lead2_id}"

company2=$(run_module "$lead2_id" "company-enrich" "company_enrich")
status2=$(echo "$company2" | jq -r '.status // empty')
echo "==> company-enrich status: ${status2}"
if [ "${status2}" != "partial" ]; then
  echo "ERROR: expected company-enrich status 'partial' for domain-only lead, got '${status2}'" >&2
  echo "$company2" >&2
  exit 1
fi
name2=$(echo "$company2" | jq -r '.fields.name // empty')
if [ -n "${name2}" ]; then
  echo "ERROR: expected empty company name for domain-only lead, got '${name2}'" >&2
  echo "$company2" >&2
  exit 1
fi
echo "$company2" | jq '{status, fields: .fields | {domain, name, website}}'

# 3. Missing permission_ref => skipped
echo "==> creating lead without permission_ref"
lead3_id=$(create_lead '{"domain":"example.com","company":"NoPerm"}')
if [ -z "$lead3_id" ]; then
  echo "ERROR: could not create lead without permission_ref" >&2
  exit 1
fi
echo "==> created lead ${lead3_id}"

company3=$(run_module "$lead3_id" "company-enrich" "company_enrich")
status3=$(echo "$company3" | jq -r '.status // empty')
reason3=$(echo "$company3" | jq -r '.reason // empty')
echo "==> company-enrich status: ${status3}, reason: ${reason3}"
if [ "${status3}" != "skipped" ]; then
  echo "ERROR: expected company-enrich status 'skipped' without permission_ref, got '${status3}'" >&2
  echo "$company3" >&2
  exit 1
fi

# 4. Lead detail returns flattened company_enrich and audit events
echo "==> fetching lead detail"
detail=$(curl -s "${BASE_URL}/api/leads/${lead_id}")
if [ "$(echo "$detail" | jq -r '.data.stage // empty')" != "enriched" ]; then
  echo "ERROR: expected stage 'enriched' for ok company-enrich lead" >&2
  echo "$detail" >&2
  exit 1
fi
audit_count=$(echo "$detail" | jq '[.data.audit_events[] | select(.module=="company-enrich")] | length')
if [ "${audit_count}" -lt 1 ]; then
  echo "ERROR: expected at least one company-enrich audit event" >&2
  echo "$detail" >&2
  exit 1
fi
echo "==> lead detail ok; ${audit_count} company-enrich audit event(s)"

echo "==> smoke-api passed"
