# Evaluation: OpenOSINT

- **Repo:** https://github.com/OpenOSINT/OpenOSINT
- **Target module:** `ai-agent-glue` / orchestration layer (Section 9 & the "AI-agent glue" row of Section 1 in [`docs/research/osint-tooling-research.md`](../docs/research/osint-tooling-research.md))
- **Evaluator:** research contributor (AI agent)
- **Date:** 2026-07-13

## 1. Summary

OpenOSINT is an AI-powered OSINT agent that puts a fleet of investigation tools behind a natural-language interface, usable as an interactive REPL, a CLI, an MCP server, or a browser Web UI ([README](https://github.com/OpenOSINT/OpenOSINT/blob/main/README.md)). The AI backend (Anthropic Claude by default, or local Ollama, or any OpenAI-compatible endpoint) decides which tools to call and chains them on findings, while the actual work is done by real underlying binaries/APIs so that "hallucinated findings are structurally impossible" ([README, header + Features table](https://github.com/OpenOSINT/OpenOSINT/blob/main/README.md)). Its most reusable asset for us is a native MCP server that exposes every tool to any MCP-compatible client (Claude Code, Claude Desktop, etc.) over stdio ([`openosint/mcp_server.py`](https://github.com/OpenOSINT/OpenOSINT/blob/main/openosint/mcp_server.py)).

## 2. License

- **License:** MIT License ([LICENSE](https://github.com/OpenOSINT/OpenOSINT/blob/main/LICENSE); confirmed via GitHub `licenseInfo` API = `mit`).
- **Commercial/internal-business use allowed without restriction?** **Yes.** The verbatim [LICENSE file](https://github.com/OpenOSINT/OpenOSINT/blob/main/LICENSE) grants rights "to use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies … without restriction." The README reiterates it is "free for any use, including personal, commercial, academic, and closed-source" ([README, License section](https://github.com/OpenOSINT/OpenOSINT/blob/main/README.md)).
- **AGPL / "no commercial use" style clauses?** **None.** Note the repo also ships [`COMMERCIAL.md`](https://github.com/OpenOSINT/OpenOSINT/blob/main/COMMERCIAL.md) / [`COMMERCIAL-LICENSE.md`](https://github.com/OpenOSINT/OpenOSINT/blob/main/COMMERCIAL-LICENSE.md) and a [`CLA.md`](https://github.com/OpenOSINT/OpenOSINT/blob/main/CLA.md); this is **not** a copyleft restriction. The "commercial license" is an *optional* paid service agreement (warranty, indemnification, SLA, priority support from €300/year) that the code explicitly states is not required for commercial use ([README, "Commercial License & Support"](https://github.com/OpenOSINT/OpenOSINT/blob/main/README.md); [`COMMERCIAL-LICENSE.md`](https://github.com/OpenOSINT/OpenOSINT/blob/main/COMMERCIAL-LICENSE.md)). MIT is clean for our internal-business use.

## 3. Maintenance health

- **Last commit:** 2026-07-10 (`480fbe6`, committed `2026-07-10T15:22:51Z`), retrieved via `gh api repos/OpenOSINT/OpenOSINT/commits`. Latest tagged release is [v2.23.0, published 2026-06-18](https://github.com/OpenOSINT/OpenOSINT/releases/tag/v2.23.0). Actively developed.
- **Open issues:** 3 open issues and 4 open PRs (GitHub search API, `type:issue`/`type:pr` + `state:open`, 2026-07-13). Very low backlog.
- **Contributors:** Effectively **single-maintainer** — real bus-factor risk. The `contributors` API shows `SonoTommy` with 256 commits, `teamvelociraptor` with 3, and `Francesco` with 1 (plus a `github-actions[bot]`). The sole maintainer is **Tommaso Bertocchi** ([README, Maintainer section](https://github.com/OpenOSINT/OpenOSINT/blob/main/README.md)). 967 stars / 143 forks give some community durability, but development is one person.
- **Release cadence:** Rapid, single-author churn. The repo is young (created [2026-05-06](https://github.com/OpenOSINT/OpenOSINT)) and is already on `v2.25.0` in [`pyproject.toml`](https://github.com/OpenOSINT/OpenOSINT/blob/main/pyproject.toml) while the newest *published* release is v2.23.0 — i.e. `main` runs ahead of releases. **Metadata drift is a mild code-smell:** the tool count is stated inconsistently across the repo — the GitHub description and [`pyproject.toml`](https://github.com/OpenOSINT/OpenOSINT/blob/main/pyproject.toml) say "16 tools", the [README](https://github.com/OpenOSINT/OpenOSINT/blob/main/README.md) says "18", and the [`mcp_server.py`](https://github.com/OpenOSINT/OpenOSINT/blob/main/openosint/mcp_server.py) docstring says "19". The installed package actually exposes 20 MCP tools (see §4).

## 4. Input / output contract

OpenOSINT can be driven three ways; the relevant contract for us is the **MCP tool interface**. The MCP server ([`openosint/mcp_server.py`](https://github.com/OpenOSINT/OpenOSINT/blob/main/openosint/mcp_server.py)) registers each capability via `@app.list_tools()` as an MCP `Tool` with a JSON-schema `inputSchema`, and dispatches calls through `@app.call_tool()` to a handler that returns a `CallToolResult` of `TextContent`. Every tool's schema is augmented with an optional `json_output` boolean; when true the text payload is structured JSON via `openosint.json_output.to_json`.

Verified by installing from PyPI (`pip install openosint`, **v2.23.0**, Python 3.14 venv) and introspecting the live server — it exposes **20 tools**: the 18 documented investigation tools plus `search_footprint` and `investigate_multi` (max 10 targets):

```
# input — enumerate the live MCP tool surface
python3 -c "import asyncio, openosint.mcp_server as m; \
  print([t.name for t in asyncio.run(m.list_tools())])"

# output (real, 2026-07-13)
MCP tool count: 20
['search_email', 'search_username', 'search_breach', 'search_whois',
 'search_ip', 'search_domain', 'generate_dorks', 'search_paste',
 'search_phone', 'search_shodan', 'search_virustotal', 'search_censys',
 'search_ip2location', 'search_abuseipdb', 'search_github', 'search_dns',
 'search_dorks_live', 'scrape_url', 'search_footprint', 'investigate_multi']
```

A single tool call and its JSON output shape (real local run of the no-network `generate_dorks` tool, which is what an MCP client receives when `json_output=true`):

```
# input — MCP call equivalent: generate_dorks(target="johndoe99", json_output=true)
python3 -c "from openosint.json_output import to_json; \
  from openosint.tools.generate_dorks import run_dork_osint; \
  print(to_json('generate_dorks','johndoe99', run_dork_osint('johndoe99')))"

# output (real, 2026-07-13, truncated)
{
  "tool": "generate_dorks",
  "target": "johndoe99",
  "timestamp": "2026-07-13T00:36:19.949673+00:00",
  "results": [
    "Google dork URLs for 'johndoe99':",
    "[+] \"johndoe99\" site:linkedin.com",
    "    https://www.google.com/search?q=%22johndoe99%22%20site%3Alinkedin.com",
    ...
  ]
}
```

**Contract shape:** input is a small typed object per tool (e.g. `{"email": "..."}`, `{"domain": "..."}`, `{"ip": "..."}`; `investigate_multi` takes `{"targets": [...]}`) — see the `inputSchema` blocks in [`mcp_server.py`](https://github.com/OpenOSINT/OpenOSINT/blob/main/openosint/mcp_server.py). Output is either a human-readable text block (default) or a uniform `{tool, target, timestamp, results[]}` JSON envelope. Errors return a `CallToolResult` with `isError=True` rather than throwing. This uniform envelope is exactly the kind of contract our [`docs/architecture.md`](../docs/architecture.md) module spec wants (namespaced output, graceful failure).

## 5. Dependencies & runtime

- **Language / runtime:** Python, `requires-python = ">=3.10"` ([`pyproject.toml`](https://github.com/OpenOSINT/OpenOSINT/blob/main/pyproject.toml)). Installed and ran cleanly on Python 3.14.
- **Install method:** `pip install openosint` from [PyPI](https://pypi.org/project/openosint/) (recommended), `pip install -e .` from source, or Docker via [`docker-compose.yml`](https://github.com/OpenOSINT/OpenOSINT/blob/main/docker-compose.yml) ([README, Installation/Docker](https://github.com/OpenOSINT/OpenOSINT/blob/main/README.md)). Core deps are heavy: `mcp`, `anthropic`, `fastapi`, `uvicorn`, `asyncpg`, `cryptography`, `reportlab`, `authlib`, etc. ([`pyproject.toml` dependencies](https://github.com/OpenOSINT/OpenOSINT/blob/main/pyproject.toml)). Several tools also require **external binaries in `PATH`** — `holehe`, `sherlock`, `sublist3r`, `phoneinfoga`; if absent the tool returns a descriptive error and the rest keep working ([README, External binaries table](https://github.com/OpenOSINT/OpenOSINT/blob/main/README.md)).
- **Required API keys / accounts:** For the **AI agent**, one of `ANTHROPIC_API_KEY`, an Ollama runtime (no key), or an OpenAI-compatible endpoint is required ([README, Configuration table](https://github.com/OpenOSINT/OpenOSINT/blob/main/README.md)). All **tool** keys are *optional* — `HIBP_API_KEY`, `SHODAN_API_KEY`, `VIRUSTOTAL_API_KEY`, `CENSYS_API_ID/SECRET`, `ABUSEIPDB_API_KEY`, `IP2LOCATION_API_KEY`, `BRIGHTDATA_API_KEY` + zone names, `GITHUB_TOKEN`, `IPINFO_TOKEN` — each gates its own tool ([README, Configuration table](https://github.com/OpenOSINT/OpenOSINT/blob/main/README.md)).
- **Expected latency for a single lookup:** Not formally benchmarked in the docs. No-network tools (`generate_dorks`, `search_dns`) return effectively instantly (local `generate_dorks` was sub-second in testing). Network tools are dominated by the upstream binary/API — `sherlock` across 300+ sites and `holehe`/`sublist3r` are seconds-to-tens-of-seconds; `--timeout N` and `--parallel` (`asyncio.gather()`) exist to bound and overlap these ([README, Features + CLI Reference](https://github.com/OpenOSINT/OpenOSINT/blob/main/README.md)).

## 6. Rate limits / ToS risk

OpenOSINT is explicitly framed "**for authorized security research only**," and its [`DISCLAIMER.md`](https://github.com/OpenOSINT/OpenOSINT/blob/main/DISCLAIMER.md) lists **prohibited** uses including "unauthorized surveillance or stalking," "harassment or targeting of private persons," and "**bypassing access controls or terms of service of third-party platforms**." Several tools carry real ToS/GDPR exposure at scale for our lead-validation use case:

- **`scrape_url` uses Bright Data Web Unlocker to bypass Cloudflare/CAPTCHA** ([README, `scrape_url`](https://github.com/OpenOSINT/OpenOSINT/blob/main/README.md)). Deliberately defeating a site's access controls to harvest data is exactly the "bypassing … terms of service" the project's own disclaimer prohibits, and is a hard PR gate under Section 11 of [`docs/research/osint-tooling-research.md`](../docs/research/osint-tooling-research.md). **Do not deploy this tool in our pipeline.**
- **`search_email` (holehe) and `search_username` (sherlock)** probe 120+ / 300+ third-party platforms per lookup ([README, Tools table](https://github.com/OpenOSINT/OpenOSINT/blob/main/README.md)). Section 11 of our research doc flags holehe-class tools as gray-zone: fine for spot-checks, risky at ad-campaign scale; must be rate-limited with a documented GDPR Art. 6 legal basis ([`docs/research/osint-tooling-research.md`](../docs/research/osint-tooling-research.md), [`docs/compliance.md`](../docs/compliance.md)).
- **`search_breach` (HaveIBeenPwned)** requires a paid `HIBP_API_KEY` and is rate-limited by HIBP itself; breach data is a *risk-score input only*, not enrichment, per our research doc.
- **`search_github`** rate limit is 60 req/h unauthenticated, 5000 req/h with a token ([README, `search_github`](https://github.com/OpenOSINT/OpenOSINT/blob/main/README.md)); `search_ip` (ipinfo.io) free tier is 50k/month; `search_dorks_live` (Bright Data SERP) bills per dork call. These are upstream provider limits, not OpenOSINT's.

Net: the **MCP/orchestration layer itself is ToS-neutral**; the risk lives in specific tools we would simply not enable.

## 7. Fit score (1-5)

**Score:** 3

**Justification:** For our platform the relevant question (per the task and Section 9 of [`docs/research/osint-tooling-research.md`](../docs/research/osint-tooling-research.md)) is whether OpenOSINT's **MCP server layer could be reused/wrapped for our modules vs. building our own**. On that axis it scores well: [`mcp_server.py`](https://github.com/OpenOSINT/OpenOSINT/blob/main/openosint/mcp_server.py) is a clean, small, MIT-licensed reference for exactly the pattern we need — a typed `list_tools()` registry, a single `call_tool()` dispatcher, a uniform `{tool, target, timestamp, results[]}` JSON envelope, graceful `isError` handling, and a multi-target fan-out (`investigate_multi`) — which maps almost directly onto the module contract in [`docs/architecture.md`](../docs/architecture.md) (namespaced output, degrade-gracefully failure mode, audit-friendly). It already speaks MCP to Claude Code/Desktop and supports Anthropic, Ollama, and OpenAI-compatible backends, answering our open question of "where the AI-agent layer sits." **But the tool *catalog* is only partially aligned with lead enrichment/validation:** it is investigation/recon-oriented (breach hunting, dorking, subdomain enum, CAPTCHA-bypass scraping), not deliverability/firmographic-oriented. Our pipeline needs SMTP/MX email deliverability (AfterShip-style), phone line-type, and company firmographics — OpenOSINT only overlaps via `search_email`/`search_phone`/`search_dns`/`search_whois`, and pairs them with tools (`scrape_url`, `search_breach`, person-pivoting `investigate_multi`) that carry the GDPR/ToS exposure flagged in Section 11. It is a strong *architectural* fit and a weak *turnkey* fit; single-maintainer bus-factor caps it at 3.

## 8. Recommendation

**Reference only** (with a targeted lean-fork of the MCP layer as the concrete follow-up).

**Reasoning:** Adopting OpenOSINT as-is would drag in an investigation-agent persona, a Bright Data CAPTCHA-bypass scraper, and breach/dorking tools that fail our compliance gates and don't serve lead validation — so **not adopt-as-is, not reject** (the MCP layer is too good to throw away). The MIT license lets us freely lift and re-wrap the parts we want.

**Concrete next step:** In Stage 2, prototype our orchestrator by *cloning the structure* of [`openosint/mcp_server.py`](https://github.com/OpenOSINT/OpenOSINT/blob/main/openosint/mcp_server.py) — its `list_tools()` schema registry + `call_tool()` dispatcher + `to_json` envelope — and register **our own** `modules/*` (email-validate, phone-validate, domain-intel, company-enrich) behind it, rather than building an MCP server from scratch. Reuse at most its low-risk, already-aligned tools (`search_dns`, `search_whois`, and `search_email`/`search_phone` under strict rate-limits with a documented [`docs/compliance.md`](../docs/compliance.md) legal basis); explicitly exclude `scrape_url`, `search_breach`, and person-pivoting `investigate_multi` from the deployed pipeline. Track this against the "adopt SpiderFoot vs. build a lightweight custom orchestrator" open question in [`docs/architecture.md`](../docs/architecture.md).

## Sources

- [OpenOSINT repository](https://github.com/OpenOSINT/OpenOSINT) — stars (967), forks (143), created 2026-05-06, default branch `main`, topics (via `gh repo view --json`).
- [README.md](https://github.com/OpenOSINT/OpenOSINT/blob/main/README.md) — feature/tool tables, interfaces (REPL/CLI/MCP/Web), AI backends, config & key table, external binaries, install, commercial-service framing, maintainer.
- [LICENSE](https://github.com/OpenOSINT/OpenOSINT/blob/main/LICENSE) — verbatim MIT text; confirmed via GitHub `licenseInfo` API (`mit`).
- [COMMERCIAL.md](https://github.com/OpenOSINT/OpenOSINT/blob/main/COMMERCIAL.md) / [COMMERCIAL-LICENSE.md](https://github.com/OpenOSINT/OpenOSINT/blob/main/COMMERCIAL-LICENSE.md) / [CLA.md](https://github.com/OpenOSINT/OpenOSINT/blob/main/CLA.md) — optional paid service agreement, not a license restriction.
- [DISCLAIMER.md](https://github.com/OpenOSINT/OpenOSINT/blob/main/DISCLAIMER.md) — acceptable/prohibited use, incl. no ToS-bypass and no targeting of private persons.
- [openosint/mcp_server.py](https://github.com/OpenOSINT/OpenOSINT/blob/main/openosint/mcp_server.py) — MCP `list_tools`/`call_tool` architecture, JSON `json_output` augmentation, `CallToolResult`/`TextContent`, `investigate_multi` (MAX_TARGETS).
- [pyproject.toml](https://github.com/OpenOSINT/OpenOSINT/blob/main/pyproject.toml) — Python `>=3.10`, dependency list, version drift (2.25.0 on `main`), "16 tools" in description.
- [Release v2.23.0](https://github.com/OpenOSINT/OpenOSINT/releases/tag/v2.23.0) — latest published release, 2026-06-18.
- [PyPI: openosint](https://pypi.org/project/openosint/) — install source; installed v2.23.0 for the real §4 runs.
- Maintenance metrics via `gh api`: latest commit `480fbe6` @ 2026-07-10T15:22:51Z; open issues = 3, open PRs = 4; contributors = SonoTommy (256), teamvelociraptor (3), Francesco (1) — retrieved 2026-07-13.
- Live tool surface (20 tools) and JSON envelope: captured by installing `openosint==2.23.0` and introspecting `openosint.mcp_server.list_tools()` / running `generate_dorks` locally, 2026-07-13.
- Cross-referenced against in-repo [`docs/research/osint-tooling-research.md`](../docs/research/osint-tooling-research.md) (Sections 1, 9, 11), [`docs/architecture.md`](../docs/architecture.md), and [`docs/compliance.md`](../docs/compliance.md).
