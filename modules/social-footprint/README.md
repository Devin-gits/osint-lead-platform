# social-footprint

Validation-stage module for the OSINT lead platform. Given a lead record, it
answers the pipeline's **Validate** question — *is this a real, active person?* —
by checking whether a handle derived from the lead is **claimed** on a curated
set of major platforms, and adds the answer to the record under a namespaced
`social_footprint` key without overwriting any raw ingested field.

It implements the module contract in
[`docs/architecture.md`](../../docs/architecture.md): partial-record in,
same-record-plus-namespaced-key out, graceful degradation, and a per-call audit
log. It is built per the Stage 1 decision
([`docs/decisions/stage-1-decision.md` → `social-footprint`](../../docs/decisions/stage-1-decision.md),
scored 4/5 in [`evaluations/maigret.md`](../../evaluations/maigret.md)):
**adopt Maigret, embedded as a Python library, as a rate-limited, per-lead,
documented-legal-basis spot check — never a bulk sweep.**

## The handle dependency (resolved internally)

Maigret validates a **username**, but the raw lead schema
(`name, email, phone, company, domain`) has no username field. The decision doc
sequenced this module *after* a handle-source exists (`email-validate` and
`domain-intel` are both merged). Rather than require a separate upstream module,
this module derives candidate handles internally as a first-class step
([`handles.go`](handles.go)):

1. **Primary — email local-part + cheap variants.** The part of `email` before
   `@`, with any `+tag` stripped, plus up to two conservative variants when the
   local-part is dotted/separated:
   - `jane.smith@acme.com` → `jane.smith`, `janesmith` (separators removed),
     `jsmith` (first-initial + last token).
   - `bob@acme.com` → just `bob` (no variants for an undotted local-part).
2. **Secondary — enriched `domain_intel.harvester` (best-effort, optional).** If
   the input is an **already-enriched** record carrying a `domain_intel.harvester`
   sub-object (the shape the merged [`domain-intel`](../domain-intel) module
   emits), this module *may* mine extra candidates from it: local-parts of
   harvester-discovered emails, then the **leading label of discovered
   hostnames** (the "hostname fragment"), skipping infrastructure labels like
   `www`/`mail`/`ns`. This composition means the module accepts either a **raw**
   lead (email only) or a **partially-enriched** one — it is purely additive and
   never required for the module to function.
3. **No usable handle → `skipped`.** If nothing derivable passes validation
   (too short, no letter, not a plausible username), the module returns
   `status: "skipped"` with a reason — *correctly nothing to check* — as opposed
   to `"unknown"`, which would mean *tried and failed*.

Candidates are checked in priority order and **capped at 3** per lead
(`MaxHandles`) to bound fan-out.

## Why a Python subprocess around an embedded library

Maigret is Python; the prior modules are Go. The cross-language integration is a
small Python wrapper ([`wrapper/maigret_check.py`](wrapper/maigret_check.py))
that **imports Maigret as a library** (its MIT license permits direct embedding —
[`evaluations/maigret.md` §2](../../evaluations/maigret.md)) and calls its async
`maigret()` entrypoint directly. The Go module invokes that wrapper as a
**subprocess**, parsing its JSON on stdout.

This is deliberate and matches the repo's existing pattern: it is the *same
subprocess boundary* [`domain-intel`](../domain-intel) uses for theHarvester, so
the module-runner stays uniform (one static Go binary per module, stdin→stdout,
no daemon). The difference from theHarvester is *why*: theHarvester's subprocess
boundary is a **license firewall** (GPL-2.0 — must not import); Maigret is MIT, so
the subprocess is only the **language bridge**, and inside it we embed the library
directly rather than shelling out to Maigret's own CLI — exactly the "adopt the
library, not the default CLI behavior" recommendation of the evaluation.

## Compliance guardrails — enforced in code, not just docs

[`evaluations/maigret.md` §6](../../evaluations/maigret.md) flags Maigret's
default behavior (fan-out to hundreds/thousands of sites, recursive pivoting onto
other people's identities, residential-proxy block-evasion) as exactly the "bulk
non-consensual scraping" pattern [`docs/compliance.md`](../../docs/compliance.md)
restricts. Every one of those is disabled here as a **code-level** control:

| Guardrail | Where enforced | What it does |
|---|---|---|
| **Curated scope** | `curatedPlatforms` in [`maigret.go`](maigret.go); re-capped in the wrapper | Only a fixed list of **15 major platforms** is ever checked — never Maigret's 3,000+ site DB. The list is a compile-time constant passed explicitly on every call; it **cannot be widened at runtime**. The wrapper additionally hard-caps at `ABSOLUTE_MAX_SITES = 30`. |
| **Handle cap** | `MaxHandles = 3` in [`socialfootprint.go`](socialfootprint.go) | At most 3 derived candidates checked per lead, bounding fan-out. |
| **Per-lead rate limit** | `rateLimiter` in [`ratelimit.go`](ratelimit.go) | An in-process limiter enforces a **minimum delay (default 5s) between consecutive per-lead checks** when a caller loops over leads reusing one `Validator`. This makes "per-lead spot check, never a bulk sweep" a runtime guarantee, not a comment. |
| **No recursion** | wrapper calls Maigret's low-level `maigret()` once, never feeding discovered IDs back in | A lead never expands into a graph of *other* people's identities. |
| **No block-evasion** | wrapper passes `proxy=None, tor_proxy=None, i2p_proxy=None, cloudflare_bypass=None` | No residential-proxy / Tor / Cloudflare-bypass ToS-circumvention. |
| **Minimal collection** | wrapper sets `is_parsing_enabled=False`; only status + URL captured | Returns a per-platform **claimed/available/unknown** signal only — **no** scraped `fullname`/`bio`/`location`/linked-accounts — matching `docs/compliance.md`'s "avoid over-collecting beyond a simple match/no-match + confidence score" rule for the social-footprint category. |
| **Documented legal basis** | `LegalBasis` logged on every audit line | `GDPR Art.6(1)(f) legitimate-interest` per `docs/compliance.md` (social footprint = "Medium" risk). |

**Why 15 platforms.** The curated set —
GitHub, GitLab, Reddit, Twitter, Instagram, Pinterest, Medium, Telegram, Keybase,
HackerNews, Steam, SoundCloud, Vimeo, About.me, Patreon — is chosen because a
claimed/unclaimed signal on these is a meaningful "real, active person" indicator
and the scale (15 requests per handle) is defensible for a per-lead spot check.
Widening scope requires editing the constant **and** re-review.

## I/O contract

- **Input (stdin):** a JSON object — a partial or partially-enriched lead record
  with at least an `email` field. Optionally a `domain_intel` sub-object (from a
  prior pipeline step) for secondary handle candidates. All fields are preserved
  untouched.
- **Output (stdout):** the same object with one key, `social_footprint`, added.

  | field | type | meaning |
  |---|---|---|
  | `status` | string | `"ok"` if at least one handle was checked; `"skipped"` if none could be derived |
  | `reason` | string | present only when `skipped` — why nothing was checkable |
  | `handles_checked` | []string | the handle strings actually checked (≤ `MaxHandles`) |
  | `handles` | []object | per-handle result blocks (see below) |
  | `active_signals` | int | total `"claimed"` hits across all handles — the headline "this identity is real/active" score |
  | `checked_at` | string | RFC3339 UTC timestamp |
  | `source_tool` | string | `soxoj/maigret@0.6.2 (embedded Python library via wrapper subprocess)` |
  | `rate_limit_note` | string | the compliance-relevant scope/rate-limit note |

  Each `handles[]` block: `handle`, `origin`
  (`email-local-part` / `email-variant` / `domain-intel-harvester`), `status`
  (`"ok"` if the Maigret run completed, `"unknown"` if it failed),
  `platforms[]` (each `{platform, status: claimed|available|unknown, url,
  http_status}`), `claimed_count`, `checked_at`, `source_tool`, and `error`
  (present only on failure).

- **Audit (stderr):** one JSON line **per handle checked** (or exactly one for a
  skip), each carrying `tool` (name + version), `checked_at`, `handle` (**the
  only PII surfaced — never the raw email/name**), `status`, and `legal_basis`.
  This satisfies the architecture "Audit" requirement: which tool/version ran,
  when, which handle was checked, and the legal-basis tag.

### Failure mode

The module never crashes the pipeline. Each handle's Maigret run is bounded by a
per-handle timeout (`SOCIAL_FOOTPRINT_TIMEOUT`, default `90s`) and wrapped in a
panic-recover; a timeout, a missing Python/Maigret install, or an unparseable
result degrades **that handle** to `status: "unknown"` with an error note while
other handles still return. A single platform timeout inside a run does not block
the others (Maigret checks platforms concurrently and reports each independently;
a blocked platform simply reports `unknown`). Exit code is `0` even on sub-check
failure; a **non-zero exit only** means stdin was not a readable JSON object.

## Run it

Install the Python dependency once (the Go binary shells out to the wrapper):

```bash
pip install -r requirements.txt          # maigret==0.6.2 (Python 3.10+)
```

Build and run:

```bash
go build -o social-footprint ./cmd/social-footprint

export SOCIAL_FOOTPRINT_WRAPPER="$PWD/wrapper/maigret_check.py"   # or install alongside the binary
echo '{"name":"Soxoj Test","email":"soxoj@example.com","company":"Social Links"}' \
  | ./social-footprint
```

Real output (stdout), from a **live** run on 2026-07-13 against Maigret 0.6.2
(abridged to a few platforms; `soxoj` is the tool author's own well-known public
handle, used here as a safe, non-private test subject — **do not run live checks
against private individuals' data**):

```json
{
  "company": "Social Links",
  "email": "soxoj@example.com",
  "name": "Soxoj Test",
  "social_footprint": {
    "status": "ok",
    "handles_checked": ["soxoj"],
    "handles": [
      {
        "handle": "soxoj",
        "origin": "email-local-part",
        "status": "ok",
        "platforms": [
          {"platform": "GitHub",   "status": "claimed",   "url": "https://github.com/soxoj",      "http_status": 200},
          {"platform": "Keybase",  "status": "claimed",   "url": "https://keybase.io/soxoj",       "http_status": 200},
          {"platform": "Medium",   "status": "claimed",   "url": "https://medium.com/@soxoj",      "http_status": 200},
          {"platform": "Patreon",  "status": "claimed",   "url": "https://www.patreon.com/soxoj",  "http_status": 200},
          {"platform": "Telegram", "status": "claimed",   "url": "https://t.me/soxoj",             "http_status": 200},
          {"platform": "GitLab",   "status": "available", "url": "https://gitlab.com/soxoj",       "http_status": 200},
          {"platform": "Reddit",   "status": "unknown",   "url": "https://www.reddit.com/user/soxoj","http_status": 403}
        ],
        "claimed_count": 5,
        "checked_at": "2026-07-13T14:35:09Z",
        "source_tool": "soxoj/maigret@0.6.2 (embedded Python library via wrapper subprocess)"
      }
    ],
    "active_signals": 5,
    "checked_at": "2026-07-13T14:35:09Z",
    "source_tool": "soxoj/maigret@0.6.2 (embedded Python library via wrapper subprocess)",
    "rate_limit_note": "per-lead rate-limited spot check (min 5s between leads); scope hard-capped to a curated 15-platform allow-list and 3 handle candidates; recursion, profile scraping, and proxy/Cloudflare block-evasion disabled; GDPR Art.6(1)(f) legitimate-interest"
  }
}
```

Audit line on stderr for the same call (one per handle; only the handle, never
the raw email):

```json
{"tool":"soxoj/maigret@0.6.2 (embedded Python library via wrapper subprocess)","checked_at":"2026-07-13T14:35:09Z","handle":"soxoj","status":"ok","legal_basis":"GDPR Art.6(1)(f) legitimate-interest"}
```

### No usable handle degrades to `skipped` (not `unknown`)

```bash
echo '{"name":"No Email","phone":"+15551234567"}' | ./social-footprint
```

```json
"social_footprint": {
  "status": "skipped",
  "reason": "no usable handle could be derived from the lead (needs an email local-part or an enriched domain_intel.harvester sub-object)",
  "handles_checked": [],
  "active_signals": 0,
  "...": "..."
}
```

### Enriched-record composition (secondary handles)

When the input already carries a `domain_intel.harvester` sub-object, extra
candidates are mined from it (best-effort, still capped at `MaxHandles`):

```bash
echo '{"email":"jane@acme.com","domain_intel":{"harvester":{"emails":["jsmith@acme.com"],"hosts":[{"host":"careers.acme.com"}]}}}' \
  | ./social-footprint
```

derives handles `jane` (primary), `jsmith` and `careers` (secondary), and checks
the first `MaxHandles` of them.

## Configuration

| Env var | Default | Meaning |
|---|---|---|
| `SOCIAL_FOOTPRINT_TIMEOUT` | `90s` | Per-handle Maigret subprocess bound (Go duration). On expiry that handle → `unknown`. |
| `SOCIAL_FOOTPRINT_MIN_INTERVAL` | `5s` | Minimum spacing between consecutive per-lead checks on one `Validator` (the rate limiter). |
| `SOCIAL_FOOTPRINT_PYTHON` | `python3` | Python interpreter to run the wrapper. |
| `SOCIAL_FOOTPRINT_WRAPPER` | *(auto)* | Path to `wrapper/maigret_check.py`. Auto-located next to the binary or under the module dir; set explicitly if installed elsewhere. |

## Test

```bash
go test ./...
```

- **Offline unit tests** (no Python, no network): `socialfootprint_test.go`
  injects a fake runner to cover the happy path (claimed-signal aggregation), the
  `skipped` degrade, per-handle `unknown` degrade on runner error, the
  `MaxHandles` fan-out cap, and the curated-scope guardrail.
- `handles_test.go` covers handle derivation (email primary + variants, `+tag`
  stripping, secondary harvester mining with infra-label exclusion, dedup, and
  `normalizeHandle` validation).
- `ratelimit_test.go` proves the limiter does not delay the first call, spaces
  the second, and honors a zero (disabled) interval.
- The CLI test (`cmd/social-footprint`) covers the full stdin→stdout contract:
  the `skipped` path fully offline, and a real **subprocess** round-trip against a
  fake wrapper script (proving JSON parse + audit-redaction without needing a live
  Maigret), plus the only non-zero-exit path (unreadable stdin).

The output shown above under **Run it** was additionally captured from a **live**
run against the real embedded Maigret 0.6.2 during development.

## Dependencies

- Go (built and tested with the toolchain pinned in `go.mod`, `go 1.24.0`).
  **No third-party Go dependencies** (standard library only).
- **Python 3.10+** with **`maigret==0.6.2`** (**MIT**), pinned in
  [`requirements.txt`](requirements.txt) — embedded as a library by the wrapper,
  not shelled out to as a CLI. Maigret needs **no API keys** for username search.
  Its site database (`data.json`) ships inside the pip package; the wrapper loads
  the bundled copy by default (no per-run auto-download).
- The module functions with **zero API keys and no paid services**.
