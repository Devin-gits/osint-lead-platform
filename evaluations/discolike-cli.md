# Evaluation: discolike-cli

- **Repo:** https://github.com/LeadGrowGTM/discolike-cli
- **Target module:** `company-enrich` (Enrich stage — firmographic/technographic enrichment)
- **Evaluator:** Devin (Stage 1 second-opinion)
- **Date:** 2026-07-19

## 1. Summary

`LeadGrowGTM/discolike-cli` is a 13-star MIT-licensed Python CLI and library for the **DiscoLike B2B company-discovery API**. It exposes commands such as `profile`, `vendors`, `score`, `growth`, and `discover` that return firmographic, technographic, growth, and digital-footprint data for a domain. The repo is a clean, typed (`pydantic` v2) client — not a standalone data source — and every call requires a `DISCOLIKE_API_KEY` plus a paid DiscoLike subscription. It is therefore **not an open-source core** for `company-enrich`, but it is a credible **optional paid adapter** if the project later wants a first-class B2B data provider behind a provider interface. It should not be the default or only path because the platform needs a cost-conscious, open-core MIT path.

## 2. License

- **License:** MIT License ([LICENSE](https://github.com/LeadGrowGTM/discolike-cli/blob/main/LICENSE); `licenseInfo.key = mit` via GitHub API).
- **Commercial/internal-business use allowed without restriction?** **Yes**, for the CLI code. MIT permits use, modification, and distribution with attribution. The DiscoLike *data* and *API*, however, are governed by the commercial DiscoLike terms and require a paid subscription/key.
- **AGPL / "no commercial use" style clauses?** **None** in the repo's own code.

## 3. Maintenance health

- **Last commit:** `pushedAt` 2026-06-24T19:59:39Z; `createdAt` 2026-03-26T12:12:35Z.
- **Open issues:** 2 (`issues.totalCount` 2).
- **Contributors:** 2 — `MitchellkellerLG` (6 commits) and `automations-lg` (1 commit). Small, agency-maintained (LeadGrow).
- **Release cadence:** No GitHub Releases observed; version in `pyproject.toml` is `0.1.0`.
- **Repository size / language:** 326 KB, Python 3.11+, MIT.
- **Test/lint setup:** `pyproject.toml` lists `pytest`, `pytest-cov`, `respx` (for httpx mocking), `ruff`, `mypy` in optional dev dependencies, and `.github/workflows/ci.yml` exists. This suggests the client is more professionally structured than `local-enrichment-tool`, but it is still early-stage.

## 4. Input / output contract

### CLI usage

```bash
pip install -e ".[dev]"
export DISCOLIKE_API_KEY="dk_..."

# Business profile for one domain
discolike profile example.com

# Tech vendors
discolike vendors example.com

# Digital footprint score
discolike score example.com
```

### Library usage (from `src/discolike/client.py`)

```python
from discolike import DiscoLikeClient

with DiscoLikeClient(api_key="dk_...") as client:
    profile = client.profile("example.com")
    vendors = client.vendors("example.com")
    score = client.score("example.com")
```

### Output fields (`reference/discolike-field-reference.md`)

`profile` returns a `BusinessProfile` with:

- `domain`, `name`, `status`, `score`
- `start_date`, `end_date`
- `address` (HQ address)
- `phones`, `public_emails`
- `domain_associations`, `social_urls`, `redirect_domain`
- `description`, `keywords`, `industry_groups`
- `employees` (employee-range bucket)

`vendors` returns a list of detected technology vendors. `score` returns a digital-footprint score (0–999) with parameter breakdown.

The Python `types.py` models use `pydantic` v2 and `extra="allow"`, so additional fields can be returned without breaking the client.

## 5. Dependencies & runtime

- **Language / runtime:** Python 3.11+ (`requires-python = ">=3.11"` in `pyproject.toml`).
- **Install method:** `pip install -e .` from the repo. Not yet published to PyPI as of this evaluation.
- **Required API keys / accounts:**
  - `DISCOLIKE_API_KEY` (required) — DiscoLike API key.
  - Paid DiscoLike subscription: Starter $99/mo, Pro $199/mo, Team $399/mo, Company $799/mo, Enterprise $1,599/mo (per `reference/discolike-field-reference.md`).
- **Key dependencies:** `httpx`, `pydantic>=2.0`, `pydantic-settings`, `click`, `rich`, `pyyaml`.
- **Expected latency for a single lookup:** `DEFAULT_TIMEOUT = 30.0` seconds with up to `MAX_RETRIES = 3` and exponential backoff (`BACKOFF_BASE = 1.0`). Real latency is dominated by the DiscoLike API round-trip; the client has built-in retry and cost tracking.
- **Cost model:** Per-query fee ($0.08–$0.18 depending on plan) + per-record fee ($1.50–$3.50 per 1,000 records). A `profile` call returns one record, so a single enrichment costs roughly the per-query fee. `CostTracker` tracks session spend and budget guards are documented at the API/account level.

## 6. Rate limits / ToS risk

- **Data source.** All data comes from DiscoLike's proprietary B2B dataset. There is no scraping of LinkedIn, Crunchbase, or other third-party sites from the CLI itself. ToS risk is concentrated in the commercial contract with DiscoLike, not in dodging platform ToS.
- **Rate limiting.** The client retries on `429` using the `Retry-After` header and exponential backoff. The actual rate limits are set by the DiscoLike plan.
- **PII / GDPR posture.** Business-profile data is company-level (name, domain, HQ, industry, employee range, public emails/phones). `contacts` command exists for B2B contact search, but that is a separate `Team+` feature and would be a separate, higher-risk concern. For a narrow `company-enrich` scope, the `profile`/`vendors`/`score` endpoints are lower-risk than contact discovery.
- **Caching.** The client has a `CacheManager` with per-endpoint TTLs. Caching enriched data must still respect DiscoLike's terms.

## 7. Fit score (1-5)

**Score:** 3

**Justification:** `discolike-cli` is the cleanest B2B-company-enrichment candidate reviewed so far. It is well-typed, MIT-licensed, has CI/test scaffolding, and exposes exactly the firmographic/technographic fields a `company-enrich` module would need. It loses points because:

1. It is a **client for a paid API**, not an open-source enrichment engine. It cannot be the default or only provider for an open-core MIT module.
2. It is **early-stage** (13 stars, 2 contributors, v0.1.0, no PyPI package yet).
3. It is **Python**, so it would be used as a subprocess or as a reference for a Go HTTP adapter, not as an imported Go library.
4. The `discover` command is designed for **lead list expansion** ("find companies like this"), which is outside the narrow scope of enriching one lead's company. Only `profile`, `vendors`, `score`, and `growth` are relevant.

For the platform's `company-enrich` module, it is a **credible optional paid adapter**, not a core.

## 8. Recommendation

**Reference / optional adapter only. Do not make it the default or sole provider.**

**Reasoning + concrete next step:** If `company-enrich` proceeds, the design should define a `Provider` interface with a no-key, deterministic `local` provider as the default and a `discolike` (or generic `disco-like`) paid adapter behind `DISCOLIKE_API_KEY`. The `discolike` adapter would call the `/profile` and `/vendors` endpoints for a lead's `domain`, map the response into the module's `Fields` struct, and produce an audit record. It should be placed **after** the local provider in a waterfall so paid calls only happen when free sources cannot satisfy the required fields. Do not import `discolike-cli` into the MIT Go core; either invoke it as a subprocess (like theHarvester/Maigret) or re-implement the small HTTP adapter in Go.

## Sources

- Repo metadata: `LeadGrowGTM/discolike-cli` — 13 stars, 2 forks, 2 contributors, created 2026-03-26, pushed 2026-06-24, 326 KB, Python, MIT license. GitHub API via `gh repo view`.
- `pyproject.toml`: Python 3.11+, dependencies `httpx`, `pydantic>=2.0`, `click`, `rich`, dev deps `pytest`, `respx`, `ruff`, `mypy` ([source](https://github.com/LeadGrowGTM/discolike-cli/blob/main/pyproject.toml)).
- `src/discolike/types.py`: Pydantic models for `BusinessProfile`, `DiscoverRecord`, `ExtractResult`, `ScoreResult`, `VendorsResult`, etc. ([source](https://github.com/LeadGrowGTM/discolike-cli/blob/main/src/discolike/types.py)).
- `src/discolike/client.py`: `DiscoLikeClient` with `profile`, `vendors`, `score`, `discover`, retry/backoff, dry-run mode, cost tracking ([source](https://github.com/LeadGrowGTM/discolike-cli/blob/main/src/discolike/client.py)).
- `reference/discolike-field-reference.md`: endpoint field reference and pricing table ([source](https://github.com/LeadGrowGTM/discolike-cli/blob/main/reference/discolike-field-reference.md)).
- `README.md`: install, quick-start, commands, cost-protection features ([source](https://github.com/LeadGrowGTM/discolike-cli/blob/main/README.md)).
- `docs/compliance.md`: platform rules on paid APIs as optional adapters and B2B data governance.
