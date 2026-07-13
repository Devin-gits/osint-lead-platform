# Evaluation: Firecrawl

- **Repo:** https://github.com/firecrawl/firecrawl
- **Target module:** `modules/extraction` (Enrich stage — turning ad landing pages / business websites into structured data)
- **Evaluator:** AI research contributor (Stage 1)
- **Date:** 2026-07-13

## 1. Summary

Firecrawl is a web-data API that turns any URL into clean, LLM-ready Markdown or structured JSON, handling JS rendering, crawling, mapping, search, and page interaction ([README](https://github.com/firecrawl/firecrawl/blob/main/README.md)). It ships as an open-source, self-hostable TypeScript service and as a managed hosted API at [firecrawl.dev](https://firecrawl.dev/?ref=github). For this platform it is the "extraction" primitive: point it at an ad landing page or a lead's business website and get back structured fields (markdown/JSON) to feed the enrichment step.

## 2. License

- **License:** GNU Affero General Public License v3.0 (AGPL-3.0). Verified against the repo's actual `LICENSE` file, which begins "GNU AFFERO GENERAL PUBLIC LICENSE Version 3" ([LICENSE](https://github.com/firecrawl/firecrawl/blob/main/LICENSE)); GitHub reports the SPDX id as `AGPL-3.0` ([license API / repo sidebar](https://github.com/firecrawl/firecrawl)). The README adds that "The SDKs and some UI components are licensed under the MIT License" ([README, License section](https://github.com/firecrawl/firecrawl/blob/main/README.md#license)).
- **Commercial/internal-business use allowed without restriction?** **Conditional.** AGPL-3.0 permits commercial and internal use, but its **Section 13 ("Remote Network Interaction")** copyleft is the key clause: if you *modify* Firecrawl and make that modified version **available to users interacting with it over a network**, you must offer those users the complete corresponding source of your modified version. See the network-interaction language in the [LICENSE file](https://github.com/firecrawl/firecrawl/blob/main/LICENSE).
  - **Does self-hosting internally trigger any obligations?** Running an **unmodified** self-hosted instance purely inside our own infrastructure (our own agents/services calling it, no external users being served the Firecrawl app itself) does **not** trigger any source-disclosure obligation — internal use and modification without conveyance/network-offering is unrestricted under AGPL. The obligation is triggered only if we (a) *modify* Firecrawl **and** (b) let outside users interact with that modified instance over a network, or otherwise distribute it; then we must publish our modifications' source under AGPL. Practically, our lead pipeline calling Firecrawl over HTTP internally is fine; exposing a *modified* Firecrawl as a customer-facing service is the risk boundary.
  - **AGPL "viral" reach into our own code:** Calling Firecrawl as a separate network service (its API) or importing only the **MIT-licensed SDK** does **not** pull our proprietary pipeline code under AGPL — AGPL copyleft attaches to the Firecrawl program and its modifications, not to independent clients that talk to it over the network/SDK. The hosted API path avoids AGPL entirely, since no covered software is conveyed to us.
- **AGPL / "no commercial use" style clauses?** **Flagged explicitly:** this is **AGPL-3.0, a strong network-copyleft license** ([LICENSE](https://github.com/firecrawl/firecrawl/blob/main/LICENSE)). There is **no** "non-commercial only" clause — commercial use is allowed — but the Section 13 network-copyleft is a real obligation if we fork-and-modify and expose it. This must be raised loudly at Stage-2 review before any modified self-host is deployed.

## 3. Maintenance health

- **Last commit:** 2026-07-11 (commit `af6d47f`, "fix(search): retry transient highlight failures (#4000)"); repo `pushedAt` 2026-07-12 ([commits API](https://github.com/firecrawl/firecrawl/commits/main)). Actively maintained (daily activity).
- **Open issues:** 45 open issues ([issues search](https://github.com/firecrawl/firecrawl/issues?q=is%3Aissue+is%3Aopen)), plus 357 open pull requests ([PR search](https://github.com/firecrawl/firecrawl/pulls?q=is%3Apr+is%3Aopen)) — the low issue count against ~150k stars indicates responsive triage; the large open-PR queue reflects heavy community contribution.
- **Contributors:** 155 ([contributors graph](https://github.com/firecrawl/firecrawl/graphs/contributors), counted via the GitHub contributors API). **Single-maintainer risk? No** — this is a company-backed (Firecrawl / Mendable) project with a large contributor base.
- **Release cadence:** Regular tagged releases; latest is **v2.11.0**, published 2026-06-19 ([releases](https://github.com/firecrawl/firecrawl/releases)). Repo created 2024-04-15, so ~2 years to v2.11 — a steady, frequent cadence.

## 4. Input / output contract

Real documented example from the README's Scrape section (no live API key was available in this environment and self-hosting requires a Docker stack, so this is the repo's own documented example, not a paraphrase) ([README §Scrape](https://github.com/firecrawl/firecrawl/blob/main/README.md#scrape)):

```
# input (Python SDK)
from firecrawl import Firecrawl
app = Firecrawl(api_key="fc-YOUR_API_KEY")
result = app.scrape('firecrawl.dev')

# equivalent raw request
curl -X POST 'https://api.firecrawl.dev/v2/scrape' \
  -H 'Authorization: Bearer fc-YOUR_API_KEY' \
  -H 'Content-Type: application/json' \
  -d '{ "url": "firecrawl.dev" }'

# output (markdown format)
# Firecrawl

Firecrawl helps AI systems search, scrape, and interact with the web.

## Features
- Search: Find information across the web
- Scrape: Clean data from any page
- Interact: Click, navigate, and operate pages
- Agent: Autonomous data gathering
```

For lead extraction the more relevant contract is **structured JSON via a schema**. The README's Agent example shows a Pydantic schema in, typed JSON out ([README §Agent with Structured Output](https://github.com/firecrawl/firecrawl/blob/main/README.md#agent-with-structured-output)):

```
# input: schema + prompt
class Founder(BaseModel):
    name: str
    role: Optional[str]
class FoundersSchema(BaseModel):
    founders: List[Founder]
result = app.agent(prompt="Find the founders of Firecrawl", schema=FoundersSchema)

# output
{
  "founders": [
    {"name": "Eric Ciarla", "role": "Co-founder"},
    {"name": "Nicolas Camara", "role": "Co-founder"},
    {"name": "Caleb Peffer", "role": "Co-founder"}
  ]
}
```

This "URL/prompt + schema → structured JSON" shape maps directly onto our need to turn a landing page into `{name, company, ...}` lead fields. Note: schema/JSON extraction is an **AI feature** — on self-host it requires an `OPENAI_API_KEY` (or an Ollama/OpenAI-compatible endpoint) per the self-host env template ([SELF_HOST.md](https://github.com/firecrawl/firecrawl/blob/main/SELF_HOST.md)).

## 5. Dependencies & runtime

- **Language / runtime:** TypeScript/Node.js service (primary language reported as TypeScript) ([repo](https://github.com/firecrawl/firecrawl)); backed by a Playwright service for JS rendering ([SELF_HOST.md](https://github.com/firecrawl/firecrawl/blob/main/SELF_HOST.md)). Official SDKs for Python, Node.js, Java, Elixir, Rust, plus a community Go SDK ([README §SDKs](https://github.com/firecrawl/firecrawl/blob/main/README.md#sdks)).
- **Install method:**
  - Hosted API: sign up at [firecrawl.dev](https://firecrawl.dev) for an `fc-` API key ([README §Quick Start](https://github.com/firecrawl/firecrawl/blob/main/README.md#quick-start)); client via `pip install firecrawl-py` or `npm install firecrawl`.
  - Self-host: **Docker**, configured via a root `.env` file (template in [SELF_HOST.md](https://github.com/firecrawl/firecrawl/blob/main/SELF_HOST.md)); full guide at [docs.firecrawl.dev/contributing/self-host](https://docs.firecrawl.dev/contributing/self-host).
- **Required API keys / accounts:**
  - Hosted: a Firecrawl account + `fc-` API key.
  - Self-host: no Firecrawl account needed; DB auth is off by default (`USE_DB_AUTHENTICATION=false`). **AI/JSON extraction** needs an `OPENAI_API_KEY` or a local Ollama/OpenAI-compatible endpoint; proxies and SearXNG-backed search are optional env config ([SELF_HOST.md](https://github.com/firecrawl/firecrawl/blob/main/SELF_HOST.md)).
- **Expected latency for a single lookup:** README claims **P95 latency of 3.4s** across millions of pages for the hosted service ([README §Why Firecrawl?](https://github.com/firecrawl/firecrawl/blob/main/README.md#why-firecrawl)). This is a vendor figure for the cloud offering; self-hosted latency will vary with proxy/Playwright setup and was not independently measured here.

## 6. Rate limits / ToS risk

- **Firecrawl behavior:** By default Firecrawl **respects `robots.txt`** and the README states plainly: *"It is the sole responsibility of end users to respect websites' policies when scraping... adhere to applicable privacy policies and terms of use"* ([README, footer](https://github.com/firecrawl/firecrawl/blob/main/README.md)). So the ToS risk is not in Firecrawl itself but in **which sites we point it at** and at what scale.
- **Self-host limitation affecting risk/scale:** Self-hosted instances **do not** get Fire-engine (the cloud's advanced IP-block/robot-detection handling); complex anti-bot targets may not work or need manual proxy config ([SELF_HOST.md, "Considerations"](https://github.com/firecrawl/firecrawl/blob/main/SELF_HOST.md)). This means the hosted API is materially more capable against defended landing pages, but using it sends target data to a third party.
- **Fit with our compliance rules:** Our [`docs/compliance.md`](../docs/compliance.md) rates "Web scraping infra (Firecrawl, Crawl4AI, Scrapy, browser-use)" as **Low** risk, with the caveat that *"the infra itself is neutral; risk depends on what target site's ToS says — check before scraping any specific ad network or business site's landing pages beyond ones we're validating with permission."* Concretely: (a) only scrape landing pages/sites we have permission to process (Hard Rule 2 — respect third-party ToS; Hard Rule 4 — log source permission); (b) if we use the **hosted API**, sending lead-related URLs/content to Firecrawl/Mendable is a **third-party processor** transfer that needs a DPA and a GDPR legal-basis entry (Firecrawl advertises SOC2 Type 2 — [SELF_HOST.md](https://github.com/firecrawl/firecrawl/blob/main/SELF_HOST.md)); (c) self-hosting keeps data in-house and avoids that transfer, at the cost of Fire-engine capabilities.

## 7. Fit score (1-5)

**Score:** 4

**Justification** (connected to `README.md`'s pipeline table): The README pipeline table lists `modules/extraction` under the **Enrich** stage with candidate tools "Firecrawl, Crawl4AI, browser-use, Scrapy," and the pipeline's first hop is literally *"Ad / Website → Raw lead (name, email, phone, company, domain)."* Firecrawl targets exactly this hop: its `scrape` + schema-driven `agent`/extract flow converts an arbitrary landing-page URL into structured JSON fields with **no per-site selector maintenance**, which is decisive for ad landing pages whose markup is inconsistent and disposable. It clears every gate our reviewer checks — actively maintained (last commit 2026-07-11, 155 contributors), production-grade, and directly on-target for the extraction primitive rather than generic praise. It loses one point because the strongest capabilities (Fire-engine anti-bot handling, and the AI JSON/extract path without bringing your own LLM key) are **cloud-only** ([SELF_HOST.md considerations](https://github.com/firecrawl/firecrawl/blob/main/SELF_HOST.md)), so the fully in-house, GDPR-cleanest deployment is the weakest variant — a real tension against our Portugal/EU data-residency posture in [`docs/compliance.md`](../docs/compliance.md).

## 8. Recommendation

**Adopt as-is** (hosted API for the Stage-2 spike), with self-host held as a documented GDPR fallback.

**Reasoning + concrete next step:**

Firecrawl is the best-fit, lowest-maintenance option for the `extraction` module: it directly implements "URL → structured lead fields," is heavily maintained, and its **MIT-licensed SDK** means integrating it does **not** impose AGPL obligations on our own pipeline code. The AGPL-3.0 network-copyleft only bites if we *fork and modify* the server *and* expose that modified instance to external users — which is not our plan — so the license is manageable but must be recorded as a Stage-2 gate.

Concrete next step for Stage 2 (do **not** build yet — this evaluation must be reviewed/approved first per `CONTRIBUTING.md`):
1. Run a time-boxed spike in `modules/extraction/` that calls the **hosted** `scrape`/`agent` endpoints with a lead-field Pydantic schema against 5–10 real (permissioned) landing pages, measuring extraction accuracy and latency vs. the vendor's 3.4s P95 claim.
2. Before that spike processes any real lead data, add a **DPA + GDPR legal-basis entry** for Firecrawl/Mendable as a third-party processor (per `docs/compliance.md` Hard Rule 4), and add a Firecrawl row to the compliance table noting the hosted-vs-self-host data-residency trade-off.
3. In parallel, note the **self-host (unmodified, Docker) path** as the fallback if the DPA/data-residency review fails — it keeps data in-house but loses Fire-engine, so anti-bot-heavy targets should be benchmarked on it separately. Keep any self-host **unmodified** to stay clear of AGPL Section 13; if modifications become necessary, budget for publishing that fork's source.

## Sources

- Repo, stars, license id, primary language, default branch, latest release: [github.com/firecrawl/firecrawl](https://github.com/firecrawl/firecrawl) and [releases](https://github.com/firecrawl/firecrawl/releases) (v2.11.0, 2026-06-19).
- License text (AGPL-3.0, network-interaction clause): [LICENSE](https://github.com/firecrawl/firecrawl/blob/main/LICENSE).
- Features, quick start, scrape/agent I/O examples, SDK list, latency claim, robots.txt/ToS statement, "Open Source vs Cloud", MIT-SDK note: [README.md](https://github.com/firecrawl/firecrawl/blob/main/README.md).
- Self-host steps, `.env` template, Fire-engine limitation, SOC2 Type 2, AI-key requirement: [SELF_HOST.md](https://github.com/firecrawl/firecrawl/blob/main/SELF_HOST.md) and [docs.firecrawl.dev/contributing/self-host](https://docs.firecrawl.dev/contributing/self-host).
- Last commit date/sha: [commits](https://github.com/firecrawl/firecrawl/commits/main) (`af6d47f`, 2026-07-11).
- Open issues (45) / open PRs (357): [issues](https://github.com/firecrawl/firecrawl/issues?q=is%3Aissue+is%3Aopen) / [pulls](https://github.com/firecrawl/firecrawl/pulls?q=is%3Apr+is%3Aopen).
- Contributors (155): [contributors graph](https://github.com/firecrawl/firecrawl/graphs/contributors).
- Pipeline stage mapping: this repo's [`README.md`](../README.md) pipeline table; risk classification: this repo's [`docs/compliance.md`](../docs/compliance.md).
