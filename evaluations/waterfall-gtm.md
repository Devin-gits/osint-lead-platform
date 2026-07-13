# Evaluation: Waterfall GTM

- **Repo:** https://github.com/codyschneiderx/waterfall-gtm
- **Target module:** `company-enrich` (Enrich stage — firmographic enrichment) — evaluated **as an architectural pattern reference for cost-conscious enrichment**, per [`docs/research/osint-tooling-research.md`](../docs/research/osint-tooling-research.md) Section 6, not as a drop-in library.
- **Evaluator:** research contributor (AI agent, Stage 1)
- **Date:** 2026-07-13

## 1. Summary

Waterfall GTM is a ~2k-LOC single-author Python reference implementation of a "waterfall" lead-enrichment pipeline: it cascades a lead (name + company domain) through an ordered list of **paid** enrichment providers (Apollo → Clearbit → ZoomInfo for company; Apollo → Hunter → Prospeo → Dropcontact for contact), stops or merges when data is found, verifies email via ZeroBounce, scores against an ICP, generates AI outreach copy, and syncs to HubSpot ([README](https://github.com/codyschneiderx/waterfall-gtm/blob/main/README.md)). It is a **small, immature repo** — 32 stars, 11 forks, one contributor, and only two commits, both from 2026‑01‑30/31, with the top of the README being a marketing splash for the author's product Graphed.com ([repo metadata via `gh`](https://github.com/codyschneiderx/waterfall-gtm), [commits](https://github.com/codyschneiderx/waterfall-gtm/commits/main)). Its value to us is therefore the **pattern** — a clean, provider-agnostic `WaterfallProcessor` abstraction ([processors/waterfall.py](https://github.com/codyschneiderx/waterfall-gtm/blob/main/processors/waterfall.py)) — rather than the code as an adoptable dependency.

## 2. License

- **License:** MIT License, Copyright (c) 2024 Cody Schneider ([LICENSE](https://github.com/codyschneiderx/waterfall-gtm/blob/main/LICENSE); confirmed `"license":{"key":"mit"}` via GitHub API).
- **Commercial/internal-business use allowed without restriction?** **Yes.** MIT permits use, copy, modify, merge, publish, distribute, sublicense, and sell without restriction, subject only to retaining the copyright + permission notice ([LICENSE](https://github.com/codyschneiderx/waterfall-gtm/blob/main/LICENSE)). If we lift any of its code, we must carry that MIT notice.
- **AGPL / "no commercial use" style clauses?** **None.** Plain, unmodified MIT — no copyleft, no network-use disclosure, no non-commercial restriction. **Important caveat:** the MIT license covers only *this repo's own code*. It grants **no rights whatsoever** over the third-party APIs it wraps (Apollo, Clearbit, ZoomInfo, Hunter, Prospeo, Dropcontact, ZeroBounce, OpenAI, HubSpot); each of those is governed by its own commercial ToS — see Section 6.

## 3. Maintenance health

- **Last commit:** **2026‑01‑31** (`915fa03` "Add Graphed.com branding and screenshot to README"; the only other commit is `8b5b3ff` "Initial commit" on 2026‑01‑30) ([commits](https://github.com/codyschneiderx/waterfall-gtm/commits/main)). **Note:** the research doc lists this repo as "Updated 2026‑06‑21" ([research doc §6](../docs/research/osint-tooling-research.md)), but the GitHub API reports `pushedAt` of `2026‑01‑31T00:17:40Z` and no commits after Jan 31 — the doc's date appears inaccurate and should not be relied on.
- **Open issues:** **0** (`hasIssuesEnabled` true, `issues.totalCount` 0; issue search `state:open` → `total_count` 0) ([issues](https://github.com/codyschneiderx/waterfall-gtm/issues)). Zero issues here reflects low adoption, not proven stability.
- **Contributors:** **1** — `codyschneiderx`, 2 contributions ([contributors API](https://github.com/codyschneiderx/waterfall-gtm/graphs/contributors)). **Single-maintainer risk: YES**, and acute — this is effectively a one-shot publish with no subsequent maintenance, tests-in-CI, or release history.
- **Release cadence:** **None.** No GitHub Releases, no tags, no PyPI package — it is not distributed as a versioned library; the only install path is `git clone` ([README installation](https://github.com/codyschneiderx/waterfall-gtm/blob/main/README.md)). Repo created 2026‑01‑30, ~383 KB, 100% Python.

## 4. Input / output contract

The reusable pattern lives in `WaterfallProcessor`. Its contract is: **in →** an ordered list of enricher objects (each implementing `enrich_company(domain)` / `enrich_contact(...)`) plus a `merge_results` flag and optional `required_fields`; **out →** a tuple `(merged model | None, [per-provider results])`. The core cascade loop is not paraphrased — it is verbatim from [processors/waterfall.py](https://github.com/codyschneiderx/waterfall-gtm/blob/main/processors/waterfall.py):

```python
for enricher in self.company_enrichers:
    result = enricher.enrich_company(domain)
    results.append(result)

    if result.success and result.company:
        sources.append(result.source)
        if merged_company is None:
            merged_company = result.company.model_copy(deep=True)
        elif self.merge_results:
            merged_company = self._merge_companies(merged_company, result.company)

        # Stop early once all required fields are satisfied
        if required_fields and merged_company:
            missing = self._get_missing_fields(merged_company, required_fields)
            if not missing:
                break
        # If not merging, stop at the first success
        if not self.merge_results:
            break
```

The provider order is data-driven from [config.yaml](https://github.com/codyschneiderx/waterfall-gtm/blob/main/config.yaml) (`waterfall.company: [apollo, clearbit, zoominfo]`). A **real, executable** demonstration of the fallback ("provider A misses → fall back to B") is the project's own test, reproduced verbatim from [tests/test_waterfall.py](https://github.com/codyschneiderx/waterfall-gtm/blob/main/tests/test_waterfall.py):

```python
# input: Apollo returns no data, Clearbit returns a hit
enrichers = [
    MockEnricher("apollo"),                                   # miss
    MockEnricher("clearbit", company_data={"name": "Acme"}),  # hit
]
processor = WaterfallProcessor(company_enrichers=enrichers, contact_enrichers=[], merge_results=False)
company, results = processor.enrich_company("acme.com")

# output (asserted by the test)
assert company.name == "Acme"       # value came from the fallback provider
assert results[0].success is False  # apollo missed
assert results[1].success is True   # clearbit answered
```

And the merge/early-stop behavior (multiple providers each contributing fields, stopping once `required_fields` are met), also verbatim from the same test file:

```python
enrichers = [
    MockEnricher("apollo",   company_data={"name": "Acme", "industry": "Tech"}),
    MockEnricher("clearbit", company_data={"employee_count": 100}),
    MockEnricher("zoominfo", company_data={"revenue": 5000000}),
]
processor = WaterfallProcessor(company_enrichers=enrichers, contact_enrichers=[], merge_results=True)
company, results = processor.enrich_company("acme.com", required_fields=["name", "industry"])
assert company.name == "Acme" and company.industry == "Tech"
assert len(results) == 1   # stopped after Apollo; zoominfo/clearbit never called → API cost avoided
```

That `len(results) == 1` assertion is the cost-control mechanism made concrete: fallback providers are only billed when the required fields are still missing.

## 5. Dependencies & runtime

- **Language / runtime:** Python (uses PEP 604 `X | Y` type unions and `list[...]` generics → **Python 3.10+**); a `pyproject.toml` is present ([tree](https://github.com/codyschneiderx/waterfall-gtm/blob/main/pyproject.toml)).
- **Install method:** **`git clone` + `pip install -r requirements.txt` only** — no PyPI package, no Docker image ([README installation](https://github.com/codyschneiderx/waterfall-gtm/blob/main/README.md)). Runtime deps: `pydantic>=2`, `pyyaml`, `httpx`, `tenacity` (retry), `python-dotenv`, `click`/`rich`/`tqdm` (CLI), `pandas`, `hubspot-api-client`, `openai`, `aiolimiter` ([requirements.txt](https://github.com/codyschneiderx/waterfall-gtm/blob/main/requirements.txt)).
- **Required API keys / accounts:** **Many, all paid.** Per [.env.example](https://github.com/codyschneiderx/waterfall-gtm/blob/main/.env.example) / [config.yaml](https://github.com/codyschneiderx/waterfall-gtm/blob/main/config.yaml): `APOLLO_API_KEY`, `CLEARBIT_API_KEY`, `ZOOMINFO_API_KEY`, `HUNTER_API_KEY`, `PROSPEO_API_KEY`, `DROPCONTACT_API_KEY`, `ZEROBOUNCE_API_KEY`, `HUBSPOT_API_KEY`, `OPENAI_API_KEY`. There is **no free/OSS enrichment path** — the tool is a thin orchestration layer over commercial APIs. (This qualifies the "cost-conscious" framing: the waterfall *reduces* spend on paid APIs but does not eliminate the requirement for them.)
- **Expected latency for a single lookup:** **Not documented and not independently measured here** (no installable package + all providers require paid keys, so a real timed run was not possible). Latency is dominated by the summed network round-trips of however many providers are hit before a result is found; `base.py` sets a **30 s per-request timeout** with `tenacity` retry (`stop_after_attempt(3)`, exponential backoff 2–10 s) ([enrichers/base.py](https://github.com/codyschneiderx/waterfall-gtm/blob/main/enrichers/base.py)), so a worst-case all-miss company cascade (3 providers × retries) can be tens of seconds.

## 6. Rate limits / ToS risk

This tool's entire job is to **chain multiple paid third-party enrichment APIs**, so the ToS surface is large and is the dominant risk — not the MIT code itself.

- **Provider rate limits are the tool's own config, not enforced ceilings.** [config.yaml](https://github.com/codyschneiderx/waterfall-gtm/blob/main/config.yaml) declares `rate_limits` per provider (e.g. Apollo/ZoomInfo 100 rpm, Clearbit 600 rpm, Dropcontact 50 rpm) and `requirements.txt` pulls in `aiolimiter`, but these are client-side throttles the operator sets — each provider's *actual* contractual limit is defined by that provider's plan, and exceeding it risks throttling or account suspension.
- **Chaining multiple providers multiplies ToS exposure.** Each vendor governs its data independently and several restrict what you may do with results — e.g. **ZoomInfo is enterprise-contract-only** (the README itself notes "Enterprise-only"), and vendors like Clearbit/Apollo commonly prohibit **caching, redistributing, or co-mingling** their data with other sources. The tool's `_merge_companies`/`_merge_contacts` methods **do exactly that** — they blend fields from Apollo + Clearbit + ZoomInfo into one persisted record and write it out to CSV and HubSpot ([processors/waterfall.py](https://github.com/codyschneiderx/waterfall-gtm/blob/main/processors/waterfall.py), [integrations/hubspot.py](https://github.com/codyschneiderx/waterfall-gtm/blob/main/integrations/hubspot.py)). Merging + persisting + syncing enriched data across providers must be checked against each provider's contract before we replicate this pattern.
- **Personal-data / GDPR overlay.** The contact waterfall resolves work emails, mobile numbers, and LinkedIn URLs for named individuals — squarely the personal data our [`docs/compliance.md`](../docs/compliance.md) governs. Any adoption of this pattern must attach a documented legal basis (consent / legitimate interest) per lead and honor deletion, independent of the upstream vendors' terms. The repo itself carries **no compliance, robots, or ToS guidance** — that discipline is entirely on us.
- **Note on the tool itself:** it does *not* scrape sites or hit anything without a key, so there is no unauthenticated-scraping ToS risk of its own kind (contrast with the extraction-module tools); the risk is purely contractual with the paid vendors.

## 7. Fit score (1-5)

**Score:** 2

**Justification** (connected to `README.md`'s pipeline table — the `Enrich (company)` row, `modules/company-enrich`, candidates "discolike-cli, local-enrichment-tool, waterfall pattern + paid APIs"): Our pipeline table explicitly lists the **"waterfall pattern + paid APIs"** as a candidate for `company-enrich`, and this repo is the canonical small example of exactly that pattern — the research doc calls it a "Directly reusable architecture for your cost-conscious enrichment pipeline" ([§6](../docs/research/osint-tooling-research.md)). As an **architecture reference** it earns real credit: `WaterfallProcessor` cleanly separates the cascade/merge/early-stop policy from provider-specific `BaseEnricher` implementations, is provider-order-driven from config, and ships tests that concretely demonstrate the cost-saving early-stop (Section 4). That is a genuinely good template for our own `company-enrich` stage. It scores **only 2**, however, because it is **not adoptable as a library**: (a) single maintainer, two commits, no releases, no PyPI/Docker, last touched 2026‑01‑31 and effectively abandoned — a bus-factor of one; (b) the README is primarily a marketing splash for the author's SaaS (Graphed.com), signalling this is a demo/lead-magnet, not a maintained OSS project; (c) it hard-wires **paid-only** providers with no OSS/free path, so it does not by itself deliver "cost-conscious" enrichment — it only optimizes spend *given* you already pay every vendor; and (d) it is tightly coupled to a specific downstream (HubSpot sync, ICP scoring, OpenAI messaging) far beyond the `company-enrich` slice we'd want. It is worth more than a 1 (the pattern and abstractions are directly instructive and MIT-clean to copy), but well short of a 3+ (nothing here is safe to depend on).

## 8. Recommendation

**Reference only** (informs our own waterfall implementation; do not add as a dependency).

**Reasoning + concrete next step:** The evidence does not support adoption or forking. There is no versioned/packaged artifact to depend on, only one maintainer with a two-commit, ~6-month-stale, marketing-oriented repo, and its providers are entirely paid APIs whose data-merging behavior needs per-vendor ToS clearance (Section 6). What *is* valuable and portable is the **design**: an ordered list of interchangeable `BaseEnricher` implementations behind a `WaterfallProcessor` that (i) stops at first success or merges, and (ii) short-circuits as soon as `required_fields` are satisfied so fallback providers are never billed unnecessarily. That MIT-licensed pattern can be lawfully re-implemented (or selectively copied with the MIT notice retained) in our own `modules/company-enrich`. **Concrete next step for Stage 2:** when `company-enrich` is greenlit, spec an implementation PR that builds our *own* `WaterfallEnricher` interface modeled on `processors/waterfall.py` and `enrichers/base.py`, but (1) leads with the OSS/free candidates from research doc §6 (discolike-cli, local-enrichment-tool) as the first waterfall tiers and only falls back to paid APIs on miss — making the waterfall genuinely cost-reducing; (2) adds the per-lead GDPR legal-basis + deletion hooks required by `docs/compliance.md`, which this repo lacks; and (3) records, per provider, whether caching/merging/redistribution is contractually permitted before any cross-provider merge is enabled. Cite this evaluation, not the upstream repo, as the Stage‑2 reference.

## Sources

- Repo metadata — MIT license (`license.key = mit`), 32 stars, 11 forks, 1 contributor, `pushedAt` 2026‑01‑31, created 2026‑01‑30, 383 KB, 100% Python, 0 open issues — GitHub API via `gh` CLI: [github.com/codyschneiderx/waterfall-gtm](https://github.com/codyschneiderx/waterfall-gtm)
- Commit history (only `8b5b3ff` 2026‑01‑30 "Initial commit" and `915fa03` 2026‑01‑31 "Add Graphed.com branding"): [commits](https://github.com/codyschneiderx/waterfall-gtm/commits/main)
- Single contributor `codyschneiderx` (2 contributions): [contributors](https://github.com/codyschneiderx/waterfall-gtm/graphs/contributors)
- Open-issue count 0: [issues](https://github.com/codyschneiderx/waterfall-gtm/issues)
- LICENSE full MIT text (Copyright (c) 2024 Cody Schneider): [LICENSE](https://github.com/codyschneiderx/waterfall-gtm/blob/main/LICENSE)
- README (pipeline diagram, waterfall explanation, Graphed.com marketing splash, install-by-clone, provider tables, estimated API costs, "Enterprise-only" ZoomInfo note): [README.md](https://github.com/codyschneiderx/waterfall-gtm/blob/main/README.md)
- Waterfall cascade / merge / required-fields early-stop logic (Section 4 code, verbatim): [processors/waterfall.py](https://github.com/codyschneiderx/waterfall-gtm/blob/main/processors/waterfall.py)
- Fallback + merge behavior demonstrated in tests (Section 4 examples, verbatim): [tests/test_waterfall.py](https://github.com/codyschneiderx/waterfall-gtm/blob/main/tests/test_waterfall.py)
- `BaseEnricher` abstraction, 30 s timeout, tenacity retry (`stop_after_attempt(3)`, exponential 2–10 s): [enrichers/base.py](https://github.com/codyschneiderx/waterfall-gtm/blob/main/enrichers/base.py)
- Concrete provider adapter (Apollo endpoints/fields): [enrichers/apollo.py](https://github.com/codyschneiderx/waterfall-gtm/blob/main/enrichers/apollo.py)
- Provider order, per-provider rate-limit config, ICP scoring weights: [config.yaml](https://github.com/codyschneiderx/waterfall-gtm/blob/main/config.yaml)
- Required paid API keys: [.env.example](https://github.com/codyschneiderx/waterfall-gtm/blob/main/.env.example)
- Dependency list: [requirements.txt](https://github.com/codyschneiderx/waterfall-gtm/blob/main/requirements.txt)
- HubSpot sync coupling (persistence of merged cross-provider data): [integrations/hubspot.py](https://github.com/codyschneiderx/waterfall-gtm/blob/main/integrations/hubspot.py)
- Internal pipeline mapping, "waterfall pattern + paid APIs" candidacy, and the "Updated 2026‑06‑21" discrepancy: [`README.md`](../README.md) pipeline table, [`docs/research/osint-tooling-research.md`](../docs/research/osint-tooling-research.md) §6
- GDPR / personal-data obligations for contact enrichment: [`docs/compliance.md`](../docs/compliance.md)
