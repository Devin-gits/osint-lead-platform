# Stage 1 Decision Doc

**Author:** Reviewer (per `CONTRIBUTING.md` §"PR review process" / `docs/research/osint-tooling-research.md` §14)
**Date:** 2026-07-13
**Status:** Stage 1 (research) closed — 12/12 priority tools evaluated and merged. This doc is the gate: no `modules/<name>/` implementation branch should open for a module until its row below says so.

## How to read this doc

Each row aggregates the merged scorecard(s) in [`evaluations/`](../../evaluations/) for one pipeline module (per [`README.md`](../../README.md)'s pipeline table). "Decision" is one of:

- **Move to Stage 2** — tool choice is settled; an implementation PR against `modules/<name>/` may open.
- **Move to Stage 2, time-boxed** — settled enough to start building, but carries a named risk that must be re-checked before/at a specific point.
- **Needs second-opinion evaluation** — no candidate cleared the bar; before building, get a second AI or human evaluation of a specific alternative named below.
- **Reject, no Stage 2 action** — no module exists (or should exist) for this yet; no code should be written against it.

## Decisions by module

### `email-validate` — Move to Stage 2

| Candidate | Fit | Rec |
|---|---|---|
| [AfterShip email-verifier](../../evaluations/email-verifier-aftership.md) | 5/5 | Adopt as-is |
| [holehe](../../evaluations/holehe.md) | 2/5 | Reference only, narrow gated exception |

**Decision:** Adopt **AfterShip email-verifier** as the sole default `email-validate` engine — deliverability/syntax/MX checks, no email sent, MIT, zero API keys, sub-10ms. holehe is **not** part of the default pipeline; it may only be invoked as a manual, rate-limited, human-triggered spot-check on individually flagged leads, never run in the automated per-lead path. **Stage 2 next step:** implementation PR for `modules/email-validate/` wrapping AfterShip's Go library (or its equivalent) behind the module contract in `docs/architecture.md`.

### `domain-intel` — Move to Stage 2, time-boxed

| Candidate | Fit | Rec |
|---|---|---|
| [web-check](../../evaluations/web-check.md) | 5/5 | Fork & modify |
| [theHarvester](../../evaluations/theharvester.md) | 4/5 | Adopt as-is (CLI subprocess only, not library import) |

**Decision:** Run **both**, not one-or-the-other — they answer different questions. web-check validates "is this an established, real business domain" (DNS/SSL/WHOIS/tech-stack); theHarvester enriches "what hosts/subdomains/contacts hang off this domain." Fork web-check directly (it's a full app, MIT). Integrate theHarvester as an **external CLI subprocess or via its `restfulHarvest` REST server only** — never import its Python modules directly, since it is GPL-2.0 and our own code is MIT; subprocess invocation is mere aggregation and avoids the copyleft trigger. **Time-box:** theHarvester's most valuable sources (Hunter, SecurityTrails, Shodan) are paywalled — re-assess after Stage 2 build whether keyless-only output is enrichment-useful enough to justify the integration, or whether it should be demoted to a manual/optional module. **Stage 2 next step:** implementation PR for `modules/domain-intel/` covering both tools, with theHarvester's source allowlist explicitly excluding breach-database modules (haveibeenpwned, dehashed, leaklookup) per the source list flagged in its evaluation.

### `phone-validate` — Move to Stage 2, time-boxed *(reviewer judgment call)*

| Candidate | Fit | Rec |
|---|---|---|
| [PhoneInfoga](../../evaluations/phoneinfoga.md) | 3/5 | Fork & modify (leaning thin-adopt) |

**Decision:** **Adopt PhoneInfoga's `local` scanner now; treat the `numverify` carrier-lookup integration as a swappable, watched dependency — do not block Stage 2 waiting for a better-maintained alternative.** Reasoning: PhoneInfoga is flagged in its own README as officially unmaintained and may be archived at any time — a real risk for a production dependency, and normally disqualifying. But the actual value splits cleanly in two: (1) the `local` scanner (number validity, country/format normalization) is free, offline, and has no third-party runtime dependency — it cannot "break" from PhoneInfoga's abandonment because it's static formatting logic we can vendor or reimplement trivially if the upstream repo disappears; (2) the carrier/line-type signal (the actual fraud-relevant part) comes from `numverify`, a third-party API PhoneInfoga merely wraps — that wrapper is thin enough to swap for a direct API integration or any other carrier-lookup provider without re-doing the evaluation. No alternative tool scored higher in Stage 1 research, and blocking the whole `phone-validate` module on finding one isn't worth the delay given the risk is contained and swappable. **Time-box condition:** re-verify PhoneInfoga's repo status (archived or not) before/at the start of any Stage 2 `modules/phone-validate/` implementation PR — if archived, build directly against `numverify` (or a successor) and skip the PhoneInfoga wrapper entirely, using only its `local` logic as design reference.

### `social-footprint` — Move to Stage 2

| Candidate | Fit | Rec |
|---|---|---|
| [Maigret](../../evaluations/maigret.md) | 4/5 | Fork & modify |

**Decision:** Adopt Maigret as the `social-footprint` validator, embedded as a Python library (MIT, async-friendly). **Important dependency to design around:** Maigret takes a *username*, but the raw lead schema (`name, email, phone, company, domain`) has no username field — it only becomes usable after an upstream step derives a candidate handle (e.g. email local-part, or a handle surfaced by theHarvester/domain-intel). Treat it as a second-stage confidence signal, not a first-touch validator. Enforce rate-limited, per-lead, documented-legal-basis invocation — never bulk. **Stage 2 next step:** implementation PR for `modules/social-footprint/`, sequenced *after* `domain-intel`/`email-validate` so a candidate handle is already available by the time this module runs.

### `extraction` — Move to Stage 2

| Candidate | Fit | Rec |
|---|---|---|
| [Crawl4AI](../../evaluations/crawl4ai.md) | 4/5 | Adopt as-is (self-hosted primary) |
| [Firecrawl](../../evaluations/firecrawl.md) | 4/5 | Adopt as-is (hosted API for the Stage 2 spike) |

**Decision:** Run Crawl4AI as the **primary, self-hosted, GDPR-cleanest** extraction engine (Apache-2.0 + a non-standard attribution clause — flagged but permissive) for turning ad landing pages/business sites into structured lead fields. Hold Firecrawl as a **secondary/fallback** option for the initial Stage 2 spike given its zero-ops hosted API, but its AGPL-3.0 license and cloud-only anti-bot/extract features are a real tension against a Portugal/EU data-residency posture — do not make Firecrawl's hosted tier a permanent dependency without a documented self-host fallback plan. **Stage 2 next step:** implementation PR for `modules/extraction/` building the interface against Crawl4AI first; keep the Firecrawl adapter behind the same interface as an option, not a requirement.

### Orchestration — Reject, no Stage 2 action *(reviewer judgment call)*

| Candidate | Fit | Rec |
|---|---|---|
| [SpiderFoot](../../evaluations/spiderfoot.md) | 2/5 | Reference only, narrow optional exception |

**Decision:** **Do not adopt SpiderFoot as the orchestration backbone, and do not spec a custom orchestrator either — there is nothing to build here yet.** Reasoning: SpiderFoot's own evaluation is decisive — its last real commit to `master` was 2023-11-05 (the repo's newer `pushed_at` reflects an unmerged Dependabot branch, not real development), and it's architecturally shaped for security-recon/attack-surface mapping (200+ modules, correlation engine tuned for threat findings), not record-oriented lead enrichment. But the answer isn't "build our own orchestrator instead" — that's premature. `docs/architecture.md`'s own open question ("adopt SpiderFoot as the backbone vs. build a lightweight custom orchestrator?") shouldn't be resolved by more OSS research; it should be resolved by *evidence from actually building modules*. With zero modules built yet, we don't know what real orchestration complexity (retries, fan-out, correlation) this pipeline needs. **Next step:** revisit this decision after `email-validate` and `domain-intel` (the two cleared modules) have working Stage 2 implementations — at that point, a simple sequential/DAG pipeline runner can likely be hand-rolled in an afternoon, informed by real module contracts instead of speculation. Do not open a research or implementation branch for orchestration before then.

### AI-agent glue / MCP layer — Needs second-opinion evaluation

| Candidate | Fit | Rec |
|---|---|---|
| [OpenOSINT](../../evaluations/openosint.md) | 3/5 | Reference only, targeted lean-fork of the MCP layer |

**Decision:** No adoption yet. OpenOSINT's MCP server layer (`mcp_server.py`) is architecturally clean and directly relevant to "where does the AI-agent layer sit" (another open question in `docs/architecture.md`), but its tool catalog is investigation/recon-shaped (breach hunting, dorking, CAPTCHA-bypass scraping) — a poor match for our deliverability/firmographic-shaped needs, and it's single-maintainer (bus-factor risk). **Next step:** this is not urgent — no module depends on it. Get a second-opinion evaluation specifically scoped to "would forking just `mcp_server.py`'s registry/dispatcher pattern (not the tool catalog) save meaningful time over hand-rolling our own MCP wrapper around the modules we're already building?" once at least one real module (`email-validate`) exists to wrap.

### `company-enrich` — Needs second-opinion evaluation

| Candidate | Fit | Rec |
|---|---|---|
| [waterfall-gtm](../../evaluations/waterfall-gtm.md) | 2/5 | Reference only |

**Decision:** No library to adopt — waterfall-gtm is a two-commit, single-maintainer demo repo with no releases, hard-wired to paid-only providers (Apollo/Clearbit/ZoomInfo/etc.), and its own evaluation corrected a stale "last updated" date in the original research doc. What's genuinely useful is the **pattern** (an ordered, provider-agnostic `WaterfallProcessor` that stops at first success and short-circuits once required fields are satisfied) — MIT-licensed and safe to reference or selectively copy with attribution. **Next step:** `company-enrich` has no adoptable OSS core at all yet (the research doc's other candidates — discolike-cli, local-enrichment-tool — were never given their own Stage 1 evaluation). Before writing any `modules/company-enrich/` code, get a second-opinion evaluation of **local-enrichment-tool** (the one candidate in the original survey with a plausible free/OSS-first path) — do not build a paid-API-only waterfall as the default, since that contradicts the "cost-conscious" goal the pattern is supposed to serve.

### Risk/breach signal (h8mail) — Reject, no Stage 2 action

| Candidate | Fit | Rec |
|---|---|---|
| [h8mail](../../evaluations/h8mail.md) | 2/5 | Reference only, narrow gated internal exception |

**Decision:** There is no risk/breach row in the README pipeline table, and this decision doc does not create one. h8mail is dormant (last real commit 2022), single-maintainer, and its core value requires paid breach-database subscriptions with their own ToS. Its evaluation also corrected the research doc's license listing (it's BSD-3-Clause, not unspecified) — a licensing non-issue, but that was never the blocker. **No Stage 2 action.** If a dedicated breach-exposure risk signal is ever prioritized, that requires first adding a pipeline row and a compliance framework entry — a scoping decision for a future stage, not a tool-adoption decision — and should prefer a single maintained keyed source over reviving h8mail's dormant codebase.

## Summary table

| Module | Decision | Confidence |
|---|---|---|
| `email-validate` | **Move to Stage 2** — AfterShip email-verifier | High |
| `domain-intel` | **Move to Stage 2** (time-boxed) — web-check + theHarvester (subprocess-only) | High |
| `phone-validate` | **Move to Stage 2** (time-boxed) — PhoneInfoga, `local` scanner now, watch archive status | Medium |
| `social-footprint` | **Move to Stage 2** — Maigret, sequenced after a handle-source exists | Medium-High |
| `extraction` | **Move to Stage 2** — Crawl4AI primary, Firecrawl secondary | High |
| Orchestration | **Reject for now** — revisit after 2 modules are built | — |
| AI-agent glue | **Needs second opinion** — narrow MCP-layer question only | — |
| `company-enrich` | **Needs second opinion** — evaluate local-enrichment-tool before any build | — |
| Risk/breach (h8mail) | **Reject** — no pipeline module exists for this | — |

## What this unblocks

Per `CONTRIBUTING.md`'s Stage 2 gate, the following may now open implementation branches under `modules/<name>/`:

- `modules/email-validate/`
- `modules/domain-intel/`
- `modules/phone-validate/`
- `modules/social-footprint/`
- `modules/extraction/`

`modules/company-enrich/` remains blocked pending a second-opinion evaluation of local-enrichment-tool. Orchestration and the AI-agent/MCP layer remain explicitly out of scope until real modules exist to inform those decisions.
