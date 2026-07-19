# License inventory — `modules/extraction`

This file documents the licensing and supply-chain posture of the `modules/extraction` Stage 2 implementation. It is a planning/implementation deliverable, not legal advice.

## Own code

| Component | License | Notes |
|---|---|---|
| `modules/extraction/**/*.go` (MIT) | MIT | Platform module code in this repository. |
| `modules/extraction/wrapper/crawl4ai_extract.py` | MIT | Thin wrapper script in this repository; imports Crawl4AI at runtime. |

## Runtime dependencies

| Component | License / Terms | How it is used | Verification |
|---|---|---|---|
| Crawl4AI (`crawl4ai` PyPI package, v0.9.2) | Apache-2.0 with attribution clause | Invoked as a subprocess CLI by `wrapper/crawl4ai_extract.py`. No source is imported, vendored, or linked into the MIT Go code. The wrapper is a separate MIT Python file that calls the installed Crawl4AI package. | License verified from upstream `https://github.com/unclecode/crawl4ai/blob/main/LICENSE` at planning time. See `requirements.txt` for the pinned version. |
| Go standard library (`net/url`, `net/http`, `net`, `encoding/json`, etc.) | BSD-style (Go license) | Used for URL parsing, HTTP client, DNS resolution, and JSON I/O in the Go code. | Distributed with Go 1.22.5. |
| Firecrawl API (optional) | Hosted service / partner terms | Used only when `EXTRACTION_BACKEND=firecrawl` and `FIRECRAWL_API_KEY` are set. The Go adapter is a thin HTTP client; no Firecrawl server code is vendored. | Users must provide their own API key and accept Firecrawl's terms. |

## Transitive dependency inventory method

Crawl4AI's transitive Python dependencies are not vendored in this repository. The production deployment method should be one of:

1. **Pinned `requirements.txt` + `pip install`** in a container/venv build step. The build process should run `pip install -r requirements.txt` and capture the resolved dependency tree with `pip freeze` or `pip install --report` for SBOM generation.
2. **Container image build** that installs `crawl4ai==0.9.2` and records the installed packages (e.g., `pip freeze > installed.txt` as a build artifact).

Before production deployment, the resolved SBOM must be reviewed for:

- GPL or other copyleft components linked into the same process as the MIT wrapper.
- Components with known security advisories.
- Components whose licenses conflict with the platform's MIT license.

## Rejected / not adopted

The following tools were evaluated and explicitly rejected for this module:

| Tool / library | Why it is not used |
|---|---|
| Photon | Not adopted; recursive crawler with broad scope beyond one permissioned landing page. |
| Crawlab | Not adopted; distributed crawling platform, out of scope for a single-page extractor. |
| `joeyism/linkedin_scraper` | Not adopted; GPL-3.0 license conflict, requires authenticated LinkedIn access, violates LinkedIn ToS. |
| `initstring/linkedin2username` | Not adopted; employee enumeration and credential-based login, out of scope. |
| `vysecurity/LinkedInt` | Not adopted; archived reconnaissance tool, no explicit license, requires credentials. |
| Any LinkedIn scraping library | Not adopted; LinkedIn scraping is excluded from production per `docs/compliance.md`. |

## No vendored GPL code

No GPL, AGPL, or other copyleft component is imported, vendored, or statically linked into the MIT Go core. Crawl4AI is invoked as a separate subprocess and its Apache-2.0 license is compatible with the platform's MIT license for this usage.
