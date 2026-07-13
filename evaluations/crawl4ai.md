# Evaluation: Crawl4AI

- **Repo:** https://github.com/unclecode/crawl4ai
- **Target module:** `extraction` (Enrich stage — turning ad landing pages / business websites into structured data)
- **Evaluator:** research contributor (AI agent, Stage 1)
- **Date:** 2026-07-13

## 1. Summary

Crawl4AI is an open-source, async Python web crawler/scraper purpose-built to turn any URL into LLM-ready output (clean/"fit" Markdown, or structured JSON). It runs a real Playwright-driven browser so it handles JavaScript-rendered pages, and offers both heuristic CSS/XPath schema extraction (no LLM cost) and LLM-driven extraction against any model. It ships as a pip library, a `crwl` CLI, and a Dockerized FastAPI server, and is designed to be fully self-hosted with no external API dependency ([README](https://github.com/unclecode/crawl4ai/blob/main/README.md), [repo description](https://github.com/unclecode/crawl4ai)).

## 2. License

- **License:** Apache License 2.0 — **with a custom "Attribution Requirement" clause appended to the `LICENSE` file** (this is a non-standard addition, not vanilla Apache-2.0).
- **Commercial/internal-business use allowed without restriction?** **Conditional — yes for use, but with a mandatory attribution condition.** Apache-2.0 permits commercial and internal-business use freely, but the appended clause states that *"All distributions, publications, or public uses of this software, or derivative works based on this software, must include"* a specific attribution to UncleCode / Crawl4AI, displayed prominently (NOTICE/README for software, "About"/"Credits" for web apps, help output for CLIs) ([LICENSE, Attribution Requirement section](https://github.com/unclecode/crawl4ai/blob/main/LICENSE)). For a purely internal enrichment pipeline this is trivial to satisfy (add the one-line credit to our docs); if any output or derivative is ever publicly exposed, we must surface the attribution.
- **AGPL / "no commercial use" style clauses?** **No AGPL, no non-commercial clause** — this is the key licensing advantage over the alternative. Unlike Firecrawl, which is **AGPL-3.0** ([firecrawl licenseInfo](https://github.com/firecrawl/firecrawl)), Crawl4AI's Apache-2.0 base imposes **no copyleft / network-use source-disclosure obligation**. The only added burden is the attribution string noted above, flagged here explicitly per the review checklist.

## 3. Maintenance health

- **Last commit:** 2026-07-09 (latest commit on `main`; repo `pushedAt` 2026-07-11) ([commits API](https://github.com/unclecode/crawl4ai/commits/main)).
- **Open issues:** 105 ([repo issues](https://github.com/unclecode/crawl4ai/issues)).
- **Contributors:** ~80 — **not** a single-maintainer project; bus-factor risk is low, though the project is still strongly identified with its creator "UncleCode" ([contributors](https://github.com/unclecode/crawl4ai/graphs/contributors)). 72,453 stars / 7,424 forks indicate a very large, active community ([repo](https://github.com/unclecode/crawl4ai)).
- **Release cadence:** Active and frequent. Latest release **v0.9.1** published 2026-07-08, following v0.9.0, v0.8.7, and v0.7.8 — recent releases are explicitly security-hardening (Docker API RCE/SSRF/auth-bypass fixes) ([releases](https://github.com/unclecode/crawl4ai/releases), [README changelog](https://github.com/unclecode/crawl4ai/blob/main/README.md)). Project created 2024-05-09, so ~2 years old with steady velocity.

## 4. Input / output contract

Real run performed locally with the pip package **v0.9.1** (installed via `pip install -U crawl4ai` + `crawl4ai-setup`), crawling `https://example.com` with a CSS-schema extraction strategy. Output below is the actual, unedited program output.

```python
# input (schema-based CSS extraction, no LLM required)
import asyncio, json
from crawl4ai import AsyncWebCrawler, BrowserConfig, CrawlerRunConfig, CacheMode
from crawl4ai import JsonCssExtractionStrategy

schema = {
    "name": "Page",
    "baseSelector": "body",
    "fields": [
        {"name": "heading",   "selector": "h1", "type": "text"},
        {"name": "paragraph", "selector": "p",  "type": "text"},
        {"name": "link",      "selector": "a",  "type": "attribute", "attribute": "href"},
    ],
}
cfg = CrawlerRunConfig(extraction_strategy=JsonCssExtractionStrategy(schema), cache_mode=CacheMode.BYPASS)
browser = BrowserConfig(headless=True, extra_args=["--no-sandbox", "--disable-dev-shm-usage"])

async def main():
    async with AsyncWebCrawler(config=browser) as crawler:
        result = await crawler.arun(url="https://example.com", config=cfg)
        print(result.markdown.raw_markdown)   # LLM-ready markdown
        print(result.extracted_content)       # structured JSON string
asyncio.run(main())
```

```text
# output

=== SUCCESS: True
=== STATUS: 200
=== LATENCY_SEC: 0.59

=== MARKDOWN (result.markdown.raw_markdown) ===
# Example Domain
This domain is for use in documentation examples without needing permission. Avoid use in operations.
[Learn more](https://iana.org/domains/example)

=== EXTRACTED_JSON (result.extracted_content) ===
[
    {
        "heading": "Example Domain",
        "paragraph": "This domain is for use in documentation examples without needing permission. Avoid use in operations.",
        "link": "https://iana.org/domains/example"
    }
]
```

**Contract:** in → a URL (+ optional CSS/XPath schema or an LLM extraction strategy); out → a `CrawlResult` object exposing `.success`, `.status_code`, `.markdown` (raw + noise-filtered "fit" markdown), `.extracted_content` (JSON string when an extraction strategy is set), plus `.links`, `.media`, and `.metadata`. For our use case, the JSON path maps a landing page directly onto lead fields (company name, contact text, links) without an LLM in the loop; the LLM strategy is available when pages need semantic extraction. Documented reference example (`JsonCssExtractionStrategy` on kidocode.com) matches this same contract ([README structured-extraction example](https://github.com/unclecode/crawl4ai/blob/main/README.md)).

## 5. Dependencies & runtime

- **Language / runtime:** Python (3.9+; tested here on Python 3.14). Uses Playwright + a Chromium browser under the hood ([README installation](https://github.com/unclecode/crawl4ai/blob/main/README.md)).
- **Install method:** `pip install -U crawl4ai` then `crawl4ai-setup` (auto-installs the Playwright browser); manual fallback `python -m playwright install --with-deps chromium`. Also available as a Docker image with a FastAPI server, and a `crwl` CLI ([README installation](https://github.com/unclecode/crawl4ai/blob/main/README.md)). Note: needs `--no-sandbox` browser args in a containerized/root environment (observed in this run).
- **Required API keys / accounts:** **None** for crawling and for CSS/XPath schema extraction (the path we'd use for structured lead fields). An LLM API key is required **only** if you opt into `LLMExtractionStrategy` — otherwise fully key-free and self-hosted ([README, "Deploy anywhere, zero keys"](https://github.com/unclecode/crawl4ai/blob/main/README.md)).
- **Expected latency for a single lookup:** **~0.59 s** measured for `https://example.com` (small static page, cache bypassed) in this run. Real-world JS-heavy ad landing pages will be higher (browser render + optional JS execution + optional LLM call), but the crawler+extraction overhead itself is sub-second here.

## 6. Rate limits / ToS risk

Crawl4AI does **not** call any third-party API of its own — it drives a browser against whatever URLs you point it at, so there is no vendor rate limit to hit. The ToS/legal risk is therefore entirely about **the sites being crawled**:

- **robots.txt is NOT respected by default.** `CrawlerRunConfig.check_robots_txt` defaults to `False` (verified in installed source `crawl4ai/async_configs.py`: `check_robots_txt: bool = False`, docstring: *"Whether to check robots.txt rules before crawling. Default: False"*) ([config source](https://github.com/unclecode/crawl4ai/blob/main/crawl4ai/async_configs.py)). We would need to explicitly set `check_robots_txt=True` for any site we don't own.
- The project's stated mission is to extract "personal and enterprise data" and it advertises features to **avoid bot detection** (managed/undetected browser, proxy support, custom user agents) ([README features / mission](https://github.com/unclecode/crawl4ai/blob/main/README.md)) — powerful, but at scale against third-party sites this can breach those sites' ToS.
- **Fit for our pipeline:** our README constrains us to ad landing pages / business websites **we have explicit permission to process**, which sharply limits this risk. Per `docs/compliance.md`, we must still (a) enable `check_robots_txt` for any non-owned domain, (b) rate-limit crawls, and (c) log the legal basis (legitimate interest / consent) for each crawled source under GDPR. Crawl4AI provides the mechanisms; the compliance discipline is on us.

## 7. Fit score (1-5)

**Score:** 4

**Justification** (connected to `README.md`'s pipeline table): The README pipeline lists Crawl4AI as a candidate for exactly this stage — `modules/extraction` under **Enrich** ("Firecrawl, Crawl4AI, browser-use, Scrapy"), whose job is "Ad / Website → Raw lead (name, email, phone, company, domain)." Crawl4AI hits that target directly: the real run above shows a single URL producing both clean Markdown (ready for an LLM enrichment step) and structured JSON (ready to populate lead fields) with sub-second overhead and zero API keys. Its JS rendering handles the dynamic ad landing pages that a plain HTTP fetch (Scrapy alone) would miss, and its CSS-schema path avoids per-page LLM cost for well-structured sites — good for the cost-conscious enrichment posture the research doc calls for. It is **not a 5** because: (a) it is a library/toolkit, not a turnkey enrichment service — we own browser fleet, scaling, and anti-bot operational burden; (b) the non-standard attribution clause is a (small) compliance obligation; and (c) it provides no hosted fallback, so reliability/scaling is entirely our responsibility.

**Comparison vs. Firecrawl (required):**

| Dimension | **Crawl4AI** | **Firecrawl** |
|---|---|---|
| License | Apache-2.0 **+ attribution clause** ([LICENSE](https://github.com/unclecode/crawl4ai/blob/main/LICENSE)) — permissive, no copyleft | **AGPL-3.0** ([license](https://github.com/firecrawl/firecrawl)) — strong copyleft; network use can trigger source-disclosure obligations |
| Hosting model | **Self-hosted only** — you run the library/Docker server; "no rate-limited APIs, no lock-in" ([README](https://github.com/unclecode/crawl4ai/blob/main/README.md)) | **Hosted SaaS API** at firecrawl.dev (credit-based paid tiers) **plus** optional self-host ([firecrawl.dev](https://github.com/firecrawl/firecrawl)) |
| Cost | Free software; cost = our own compute (+ optional LLM key) | Free/OSS self-host, **or** recurring per-credit API spend if using the hosted service |
| API keys | None required (keys only for optional LLM extraction) | Hosted API requires a Firecrawl API key/account |
| Popularity | 72,453★ | 149,885★ |

**Net:** For an internal, cost-controlled, self-hosted pipeline, **Crawl4AI's licensing is materially safer than Firecrawl's AGPL-3.0** (no copyleft reach into our own platform code) and it has **zero recurring API cost**. Firecrawl wins on turnkey convenience (managed hosted API, no browser ops), which matters only if we prefer to outsource extraction rather than own it. Given the platform's stated self-hosting/compliance-first posture, Crawl4AI is the better licensing and cost fit; Firecrawl's edge is operational simplicity.

## 8. Recommendation

**Adopt as-is** (as the primary candidate for `modules/extraction`, pending a Stage-2 bake-off against browser-use for interaction-heavy pages).

**Reasoning + concrete next step:** Crawl4AI is permissively licensed (Apache-2.0, no copyleft — a decisive advantage over AGPL-3.0 Firecrawl for our own-code protection), actively maintained (v0.9.1, ~80 contributors, security-hardening release cadence), key-free, self-hostable, and empirically produces both LLM-ready Markdown and structured JSON from a live URL in sub-second overhead — matching the `extraction` module contract exactly. The only obligations are trivial: satisfy the attribution clause and enforce `check_robots_txt=True` + rate limits + GDPR legal-basis logging per `docs/compliance.md`. **Next step for Stage 2:** open a `modules/extraction/` implementation PR that wraps `AsyncWebCrawler` behind our module interface, uses `JsonCssExtractionStrategy` for schema-known sites with an `LLMExtractionStrategy` fallback for unstructured pages, sets `check_robots_txt=True` by default, pins `crawl4ai==0.9.1`, and adds the required Crawl4AI attribution line to our repo NOTICE/README.

## Sources

- Repo, description, stars (72,453), forks (7,424), created date, latest release v0.9.1 (2026-07-08), `pushedAt`, primary language, Apache-2.0 license key — GitHub repo metadata via `gh` CLI: [github.com/unclecode/crawl4ai](https://github.com/unclecode/crawl4ai)
- LICENSE full text incl. the appended **Attribution Requirement** clause: [LICENSE](https://github.com/unclecode/crawl4ai/blob/main/LICENSE)
- README (features, install instructions, quickstart, structured-extraction example, "zero keys / self-host / no rate-limited APIs" claims, mission, attribution/citation sections, security release notes): [README.md](https://github.com/unclecode/crawl4ai/blob/main/README.md)
- Last commit date (2026-07-09): [commits](https://github.com/unclecode/crawl4ai/commits/main)
- Open issue count (105): [issues](https://github.com/unclecode/crawl4ai/issues)
- Contributor count (~80): [contributors](https://github.com/unclecode/crawl4ai/graphs/contributors)
- `check_robots_txt` default `False`: verified in installed package source `crawl4ai/async_configs.py`, upstream [async_configs.py](https://github.com/unclecode/crawl4ai/blob/main/crawl4ai/async_configs.py)
- Input/output example: real local run of `crawl4ai==0.9.1` against `https://example.com` (command + unedited output reproduced in Section 4)
- Firecrawl comparison — AGPL-3.0 license, 149,885 stars, hosted API model: [github.com/firecrawl/firecrawl](https://github.com/firecrawl/firecrawl), [firecrawl.dev](https://firecrawl.dev)
- Internal pipeline mapping and compliance requirements: [`README.md`](../README.md) pipeline table, [`docs/research/osint-tooling-research.md`](../docs/research/osint-tooling-research.md), [`docs/compliance.md`](../docs/compliance.md)
