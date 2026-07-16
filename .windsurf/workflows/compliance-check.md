---
description: Compliance and license checklist before merging module changes
---

# Compliance check

Run before merging any PR that adds or changes a data source under `modules/`.

## Steps

1. Read `docs/compliance.md` and `docs/codemap/07-compliance-and-license.md`.

2. Hard-rule scan — reject if the change enables any of these in the **default** path:
   - LinkedIn profile scraping
   - Reverse-image / GHunt-style deep account discovery
   - Bulk holehe / h8mail / breach DB sweeps
   - Maigret full site DB or recursion/proxy/profile scraping
   - Import of GPL tools (theHarvester, PhoneInfoga) into Go modules

3. For each new/changed source, verify:
   - [ ] Entry in compliance risk table (or explicit N/A note)
   - [ ] Legal basis stated (usually Art. 6(1)(f) legitimate-interest)
   - [ ] Rate-limit / scope note if third-party platforms are queried
   - [ ] Audit log includes tool@version, timestamp, status, legal_basis
   - [ ] PII handling: phone redacted in audit; social uses handle not email

4. License boundary check:
   - [ ] MIT-compatible imports only for linked libraries
   - [ ] GPL tools only as external subprocess (mere aggregation) if used at all
   - [ ] AGPL (e.g. Firecrawl) not made a permanent hard dependency without self-host plan

5. Module contract check (`docs/codemap/01-module-contract.md`):
   - [ ] Namespaced key only; raw fields preserved
   - [ ] Failures → `unknown`/`skipped`, exit 0
   - [ ] Timeout + panic recover present

6. Output a short pass/fail report with any required fixes before merge.
