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

Unlike other modules, `Check` takes the **whole lead map** (`email` or `username` + optional `domain_intel`).

## Handle derivation (`handles.go`)

Priority order, deduped, then capped at `MaxHandles`:

1. **direct `username`** (optional): an explicit handle supplied by a caller or the CLI `--username` flag.
2. **email local-part** (`jane.smith@x` → `jane.smith`); strip `+tag`
3. **email variants** (if dotted/separated): `janesmith`, `jsmith`
4. **domain_intel.harvester** (optional): email local-parts from discovered emails; leading hostname labels excluding `infraLabels`

`normalizeHandle` strips common copy-paste noise: leading `@`, `http(s)://` / `www.` prefixes, trailing query strings, and final path segments. Only letters, digits, `.`, `_`, and `-` are kept; handles under 2 characters or with no letter are rejected.

No usable handle → top-level `status: "skipped"` (not `unknown`).

Origins: `direct` | `email-local-part` | `email-variant` | `domain-intel-harvester`.

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
| `source_tool`, `checked_at` | backend-dependent: Maigret / Sherlock / both / Osintgram |
| `instagram` | (Osintgram backend only) minimal enrichment on the Instagram `PlatformSignal` — counts and boolean flags; no bio/contact strings |

## CLI

```bash
cd modules/social-footprint
make pydeps                       # pip install -r requirements.txt
make build                        # go build -o bin/social-footprint ./cmd/social-footprint

export SOCIAL_FOOTPRINT_WRAPPER="$PWD/wrapper/maigret_check.py"  # optional; auto-located

echo '{"email":"soxoj@example.com"}' | ./bin/social-footprint
./bin/social-footprint --username soxoj --timeout 60s
./bin/social-footprint --email jane.smith@acme.com --timeout 60s
./bin/social-footprint --username soxoj --email jane.smith@acme.com --timeout 60s
```

Flags augment/override the `stdin` JSON record:

- `--username HANDLE` — explicit handle to check; injected as `lead["username"]` and takes priority over email-derived handles.
- `--email ADDRESS` — injected as `lead["email"]`; overrides or supplies the email field from `stdin`.
- `--timeout DURATION` — per-handle subprocess timeout; when set, overrides `SOCIAL_FOOTPRINT_TIMEOUT`.

When run from a terminal (no pipe on `stdin`), an empty/missing `stdin` is allowed and the flags populate the lead record. At least one of `--username`, `--email`, or a lead record on `stdin` must be provided.

| Env | Default | Meaning |
|-----|---------|---------|
| `SOCIAL_FOOTPRINT_TIMEOUT` | `90s` | Per-handle subprocess bound |
| `SOCIAL_FOOTPRINT_MIN_INTERVAL` | `5s` (`15s` for `osintgram` if unset) | Rate limiter spacing |
| `SOCIAL_FOOTPRINT_PYTHON` | `python3` | Interpreter |
| `SOCIAL_FOOTPRINT_WRAPPER` | auto-locate | Path to maigret_check.py |
| `SOCIAL_FOOTPRINT_BACKEND` | `maigret` | `maigret`, `sherlock`, `both`, or `osintgram` |
| `SOCIAL_FOOTPRINT_OSINTGRAM_HOME` | *(none)* | Path to separate Osintgram checkout with `main.py` |
| `SOCIAL_FOOTPRINT_OSINTGRAM_WRAPPER` | auto-locate | Path to `wrapper/osintgram_check.py` |
| `SOCIAL_FOOTPRINT_OSINTGRAM_PYTHON` | `python3` / `SOCIAL_FOOTPRINT_PYTHON` | Interpreter for Osintgram wrapper |
| `HIKERAPI_TOKEN` | *(none)* | Optional HikerAPI token for headless Osintgram runs |
| `SOCIAL_FOOTPRINT_OSINTGRAM_CREDENTIALS` | *(none)* | Optional path to `credentials.ini`; otherwise Osintgram reads `config/credentials.ini` in `OSINTGRAM_HOME` |

## Tests

```bash
cd modules/social-footprint && go test ./...
```

Offline with fake runner; handles/ratelimit unit tests; CLI uses fake wrapper script.

The Osintgram wrapper is exercised with a fake `main.py` fixture: command allowlist,
missing home, not-found exit, and a synthetic `*_info.json` parse path. Real
Instagram checks are skipped under `-short` and behind
`SOCIAL_FOOTPRINT_LIVE_OSINTGRAM=1`.

## Osintgram backend (optional Instagram depth)

`SOCIAL_FOOTPRINT_BACKEND=osintgram` enables a single-platform Instagram check via
[Datalux/Osintgram](https://github.com/Datalux/Osintgram), invoked through
`wrapper/osintgram_check.py` as a subprocess CLI.

- **GPL firewall:** the wrapper is MIT platform code; it never imports Osintgram
  source. Osintgram must be installed separately and pointed to with
  `SOCIAL_FOOTPRINT_OSINTGRAM_HOME`.
- **Command allowlist:** only `info` is permitted; the wrapper rejects
  `followers`, `fwersemail`, `photos`, `addrs`, etc.
- **Minimal collection:** the wrapper normalizes Osintgram's `info` JSON to
  counts and boolean flags only. Biography text, full contact strings, HD images,
  GPS/addresses, and follower/following graphs are dropped.
- **Rate limit:** default `SOCIAL_FOOTPRINT_MIN_INTERVAL` is `15s` for this backend
  when the env var is unset.
- **Audit:** stderr JSON lines contain only the handle and `source_tool`; tokens,
  credentials, and Instagram contact data are never logged.

## Do not

- Run Maigret's full 3000+ site DB
- Bulk-loop without rate limiter (reuse one Validator for spacing)
- Scrape profile bios/locations
- Put raw email on audit lines
- Treat as first-touch validator without a handle source
- Import or vendor Osintgram GPL source into platform code
- Enable Osintgram commands other than `info`
