# social-footprint

**Path:** `modules/social-footprint/`  
**Package:** `github.com/Moyeil-73/osint-lead-platform/modules/social-footprint`  
**Import alias:** `socialfootprint`  
**Pipeline stage:** Validate (second-stage confidence)  
**Result key:** `social_footprint`  
**Decision:** Maigret embedded as Python library; rate-limited per-lead spot check — never bulk.

## Purpose

Is this a real, active person? Claimed/available signals on a **curated** platform list for handles derived from the lead.

## Public API

```go
const LegalBasis = "GDPR Art.6(1)(f) legitimate-interest"
const SourceTool = "soxoj/maigret@0.6.2 (embedded Python library via wrapper subprocess)"
const DefaultTimeout = 90 * time.Second   // per handle
const DefaultMinInterval = 5 * time.Second
const MaxHandles = 3

func NewValidator(timeout, minInterval time.Duration) *Validator
func (v *Validator) Check(lead map[string]interface{}) (Result, []AuditRecord)
```

Unlike other modules, `Check` takes the **whole lead map** (needs email + optional domain_intel).

## Handle derivation (`handles.go`)

Priority order, deduped, then capped at `MaxHandles`:

1. **email local-part** (`jane.smith@x` → `jane.smith`); strip `+tag`
2. **email variants** (if dotted/separated): `janesmith`, `jsmith`
3. **domain_intel.harvester** (optional): email local-parts from discovered emails; leading hostname labels excluding `infraLabels`

`normalizeHandle` strips common copy-paste noise: leading `@`, `http(s)://` / `www.` prefixes, trailing query strings, and final path segments. Only letters, digits, `.`, `_`, and `-` are kept; handles under 2 characters or with no letter are rejected.

No usable handle → top-level `status: "skipped"` (not `unknown`).

Origins: `email-local-part` | `email-variant` | `domain-intel-harvester`.

## Compliance guardrails (code-enforced)

| Guardrail | Where |
|-----------|--------|
| 15 curated platforms only | `curatedPlatforms` in `maigret.go` |
| Max 3 handles | `MaxHandles` |
| Min delay between leads + exponential backoff on consecutive runner errors | `rateLimiter` (`DefaultMinInterval` 5s, capped at 60s) |
| No recursion / profile parse / proxy / Tor / CF bypass | `wrapper/maigret_check.py` |
| Audit subject = handle only | `AuditRecord.Handle` |

**Curated platforms:** GitHub, GitLab, Reddit, Twitter, Instagram, Pinterest, Medium, Telegram, Keybase, HackerNews, Steam, SoundCloud, Vimeo, About.me, Patreon.

Widening scope requires code change + compliance re-review.

## Maigret integration

- Go → subprocess → `wrapper/maigret_check.py` → imports Maigret **as library** (MIT OK)
- Subprocess is language bridge (unlike theHarvester, not a license firewall)
- Pluggable `maigretRunner` interface for offline tests
- Wrapper JSON: `results[]` with platform, status (`claimed`|`available`|`unknown`), url, http_status

## Result fields

| Field | Notes |
|-------|-------|
| `status` | `ok` if ≥1 handle checked; `skipped` if none |
| `reason` | when skipped |
| `handles_checked` | strings |
| `handles[]` | per-handle blocks + `platforms[]`, `claimed_count` |
| `active_signals` | total claimed hits (headline score) |
| `confidence` | `0.0`–`1.0` ratio of `active_signals` to the theoretical maximum (rounded to 3 decimals) |
| `metadata` | non-PII runtime/config snapshot: `source_tool`, `legal_basis`, `max_handles`, `platform_count`, `rate_limit_base`, `rate_limit_current` |
| `rate_limit_note` | compliance text embedded in every result |
| `source_tool`, `checked_at` | |

## CLI

```bash
cd modules/social-footprint
pip install -r requirements.txt   # maigret==0.6.2
go build -o social-footprint ./cmd/social-footprint
export SOCIAL_FOOTPRINT_WRAPPER="$PWD/wrapper/maigret_check.py"
echo '{"email":"soxoj@example.com"}' | ./social-footprint
```

| Env | Default | Meaning |
|-----|---------|---------|
| `SOCIAL_FOOTPRINT_TIMEOUT` | `90s` | Per-handle subprocess bound |
| `SOCIAL_FOOTPRINT_MIN_INTERVAL` | `5s` | Rate limiter spacing |
| `SOCIAL_FOOTPRINT_PYTHON` | `python3` | Interpreter |
| `SOCIAL_FOOTPRINT_WRAPPER` | auto-locate | Path to maigret_check.py |

## Tests

```bash
cd modules/social-footprint && go test ./...
```

Offline with fake runner; handles/ratelimit unit tests; CLI uses fake wrapper script.

## Do not

- Run Maigret's full 3000+ site DB
- Bulk-loop without rate limiter (reuse one Validator for spacing)
- Scrape profile bios/locations
- Put raw email on audit lines
- Treat as first-touch validator without a handle source
