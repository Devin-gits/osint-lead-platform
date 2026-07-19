# Risk scoring v2 — deterministic score and level bands

**Status:** implemented in `services/control-plane` and `ui/web-console`  
**Date:** 2026-07-19

## Goal

Replace ad-hoc risk level derivation with a documented, deterministic
`risk_score` (0–100) and `risk_level` (`low`, `medium`, `high`, `unknown`)
computed from module results and lead fields. No ML, no paid reputation APIs.

## Output

- `risk_score`: integer 0–100 (higher = riskier).
- `risk_level`:
  - `low` when `risk_score <= 33`
  - `medium` when `34 <= risk_score <= 66`
  - `high` when `risk_score >= 67`
  - `unknown` when no risk signals are available at all

## Scoring formula

Start at 0, apply additive signals, clamp to `[0, 100]`. Negative modifiers
can reduce the score but never below 0.

| Signal | Points | Condition |
|---|---:|---|
| `email_validate` error | +40 | `status == "error"` |
| `email_validate` disposable flag | +15 | `is_disposable == true` (max +30 across email flags) |
| `email_validate` role account flag | +15 | `is_role_account == true` |
| `email_validate` free-provider flag | +15 | `is_free_provider == true` |
| `email` present but `email_validate` not run | +10 | `email` non-empty and no result |
| `phone_validate` error | +35 | `status == "error"` |
| `phone_validate` ok but invalid/impossible | +25 | `status == "ok"` and (`is_valid_number == false` or `is_possible == false`) |
| `phone` present but `phone_validate` not run | +10 | `phone` non-empty and no result |
| `domain_intel` error | +20 | `status == "error"` |
| `domain_intel` hard fail signals | +10 | `status == "ok"` and (`resolvable == false` OR `ssl.valid == false` OR `http.status_code >= 400`) |
| `social_footprint` error | +10 | `status == "error"` |
| `company_enrich` or `extraction` error | +10 | either `status == "error"` |
| All required CRM contact checks green | −10 | required channel (`email` or `phone`) has `*_validate.status == "ok"` |
| `company_enrich` ok or `extraction` ok | −5 | either `status == "ok"` |

The email flag bonus is capped at +30 (e.g. disposable + role = 30, all three
flags still 30).

## No-signals case

If the lead has no `email`, `phone`, `domain`, `url`, or module results, the
score is `0` and the level is `unknown`. A lead with only `company` and no
module results is treated as having no signals and is therefore `unknown`.
This is the only case that produces `unknown`.

## When recomputed

- After every successful `POST /api/leads/{id}/run` and per-lead batch pipeline
  run.
- Persisted to the lead record so list and detail views stay consistent.
- `GET /api/leads/{id}/risk` returns a fresh, on-demand report with the score,
  level, and contributing factors.
- `crm_ready` promotion continues to block on `risk_level == "high"` (see
  `docs/decisions/crm-ready-policy.md`).

## Examples

| Lead fixture | Expected risk |
|---|---|
| email ok, syntax valid, MX present, company_enrich ok | low (score 0 after green contact −10 and company ok −5, clamped at 0) |
| email ok but `is_disposable: true` | medium/high depending on remaining flags |
| email_validate error | high (+40) |
| phone present, never validated | low/medium (+10 from unvalidated contact) |
| company only, no module results | unknown (score 0) |

## Implementation notes

- Implemented in `services/control-plane/internal/risk`.
- Used by `services/control-plane/internal/runner` after module results are
  merged.
- Constants are named and exported for tests.
- No module behaviour changes; the risk package only reads existing result
  fields.

## Residuals

- Weight tuning is governance-driven; v3 could expose configurable weights.
- Async recompute on upstream module data changes is not implemented.
