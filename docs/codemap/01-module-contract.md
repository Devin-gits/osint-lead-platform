# Module contract

Canonical contract: `docs/architecture.md` (proposed) + patterns in all four implemented modules.

## I/O shape

| Stream | Content |
|--------|---------|
| **stdin** | One JSON object = partial lead record |
| **stdout** | Same object + one namespaced result key (pretty-printed JSON) |
| **stderr** | One or more JSON audit lines (one object per line) |

Decode lead as `map[string]interface{}` so **all raw fields are preserved**. Never overwrite ingested fields; only add the namespaced key.

## Namespaced result keys

| Module | Key |
|--------|-----|
| email-validate | `email_validate` |
| domain-intel | `domain_intel` |
| phone-validate | `phone_validate` |
| social-footprint | `social_footprint` |

## Status semantics

| Status | Meaning |
|--------|---------|
| `ok` | Check ran successfully (may still include weak signals, e.g. deliverable unknown) |
| `unknown` | Attempted but failed (timeout, tool missing, parse error) — **not** a crash |
| `skipped` | Correctly nothing to do (e.g. no handle, no API key for optional path) |

Top-level and sub-tool statuses may differ (e.g. domain-intel web_check `ok` + harvester `unknown`).

## Exit codes

| Code | When |
|------|------|
| `0` | Well-formed JSON lead read and a record emitted — **including** unknown/skipped results |
| non-zero | stdin unreadable or not valid JSON |

Library APIs (`Validate` / `Analyze` / `Check`) **never return errors for tool failure**; they return in-band status.

## Audit record (stderr)

Every module emits structured audit JSON. Common fields:

- `tool` — name + version string
- `checked_at` — RFC3339 UTC
- `status` — outcome
- `legal_basis` — almost always `GDPR Art.6(1)(f) legitimate-interest`
- Subject field varies:
  - email-validate: `email` (raw)
  - domain-intel: `domain`
  - phone-validate: `phone` (**redacted**, e.g. `+14*******86`)
  - social-footprint: `handle` only (never raw email/name)

Audit is written **before** stdout encode so a log line exists even if encode fails.

Multi-audit modules:

- domain-intel: **2** lines (web_check + harvester)
- phone-validate: **2** lines (local + numverify)
- social-footprint: **1 per handle** checked, or 1 for skip
- email-validate: **1** line

## Lead schema (raw)

Expected ingest fields (not all required for every module):

```
name, email, phone, company, domain, source_id, permission_ref
```

Modules only **require** the field they operate on:

| Module | Minimum field |
|--------|---------------|
| email-validate | `email` |
| domain-intel | `domain` (URL or bare domain; normalized) |
| phone-validate | `phone` (prefer E.164 with `+`) |
| social-footprint | `email` (or enriched `domain_intel.harvester`) |

## Package + CLI split

```
modules/<name>/
  <pkg>.go              # library: NewX, Validate/Analyze/Check, Result, AuditRecord
  cmd/<name>/main.go    # CLI: read stdin, call library, audit stderr, write stdout
  *_test.go             # package tests
  cmd/<name>/main_test.go
  go.mod                # independent Go module
  README.md
```

Go module path pattern:

```
github.com/Moyeil-73/osint-lead-platform/modules/<name>
```

## Failure mode checklist (must hold for new modules)

- [ ] Timeout bounded (env override + default)
- [ ] Panic recover around external/tool calls
- [ ] Missing input → unknown/skipped + error note, not panic
- [ ] Optional external binary/API missing → degrade, don't fail process
- [ ] Concurrent sub-tools independent (if multi-source)
- [ ] LegalBasis constant logged on every audit line
