# OSINT Tooling Research — Lead Enrichment & Validation Platform

**Purpose:** Compile the strongest open-source OSINT building blocks on GitHub to power a platform that **enriches and validates leads generated from ads and business websites** (data collected with explicit permission). This document is the output of Stage 1 (research only) and doubles as the brief for Stage 2 (build), including the repo structure, agent task prompts, and my PR review checklist as the designated reviewer.

All repos below were pulled live from GitHub search (stars, last update, license) on 2026-07-13. "★" = GitHub stars, "Updated" = last push date.

---

## 1. How the categories map to your use case

Your pipeline is effectively: **Ad/Website → raw lead (name, email, phone, company, domain) → enrichment (fill gaps) → validation (is this real / deliverable / low-risk) → CRM-ready record.** The tool categories below map directly onto that flow:

| Pipeline stage | Category | Best OSS building blocks |
|---|---|---|
| Ingest | Website/domain intelligence | web-check, theHarvester |
| Enrich (company) | Firmographic enrichment | discolike-cli, local-enrichment-tool + paid APIs (Clearbit/Apollo/PDL) |
| Enrich (contact) | Email/phone discovery | theHarvester, PhoneInfoga |
| Validate (email) | Deliverability + real-account check | AfterShip email-verifier, holehe |
| Validate (identity/risk) | Breach exposure, fraud signal | h8mail, MailAccess |
| Validate (person is real) | Social footprint check | Sherlock, Maigret, Social-Analyzer |
| Extraction infra | Pulling data off ad landing pages/sites | Firecrawl, Crawl4AI, Playwright/browser-use, Scrapy |
| Orchestration | Tying modules together | SpiderFoot, Recon-ng, sn0int |
| AI-agent glue | Letting your agents call tools natively | MCP-based OSINT servers |

---

## 2. Domain & website intelligence (validate the "website we have")

| Repo | ★ | Updated | License | Why it matters |
|---|---|---|---|---|
| [lissy93/web-check](https://github.com/lissy93/web-check) | 34,130 | 2026-07-12 | MIT | All-in-one website analyzer (DNS, SSL, headers, tech stack, WHOIS, social tags, blacklist status) in one open-source app — ideal as the core "validate this lead's website" module, and it's a full web app you can fork directly. |
| [laramies/theHarvester](https://github.com/laramies/theHarvester) | 16,765 | 2026-07-12 | GPL-family | Pulls emails, subdomains, names, and employee data tied to a domain from public sources (search engines, Shodan, crt.sh) — good first pass to auto-enrich a company domain into contacts. |
| [projectdiscovery/subfinder](https://github.com/projectdiscovery/subfinder) | 13,997 | 2026-07-12 | MIT | Fast, actively-maintained subdomain enumeration — useful for mapping a business's full web footprint from one root domain (find their real store/landing pages vs. throwaway ad pages). |
| [Security-Tools-Alliance/rengine-ng](https://github.com/Security-Tools-Alliance/rengine-ng) | 177 | 2026-07-09 | GPL-3.0 | Full automated recon **platform** (not just a script) with a UI and scheduled scans — a good architectural reference for how to structure your own dashboard. |
| [zmh-program/next-whois](https://github.com/zmh-program/next-whois) | 583 | 2026-07-10 | MIT | Modern WHOIS/domain-age lookup with a clean UI — useful signal for "is this a brand-new throwaway domain or an established business" fraud check. |

## 3. Email discovery, verification & validation

| Repo | ★ | Updated | License | Why it matters |
|---|---|---|---|---|
| [AfterShip/email-verifier](https://github.com/AfterShip/email-verifier) | 1,588 | 2026-07-08 | MIT | Production-grade Go library: syntax, MX, SMTP deliverability, disposable/free-provider/role-account detection — no emails actually sent. This is the strongest OSS candidate for your core "is this email real and deliverable" validator. |
| [megadose/holehe](https://github.com/megadose/holehe) | 11,656 | 2026-07-12 | GPL-3.0 | Checks whether an email is registered on 120+ platforms (without sending resets) — a strong secondary validation signal ("this lead's email is tied to a real, active online identity"). |
| [KatrielMoses/MailAccess](https://github.com/KatrielMoses/MailAccess) | 203 | 2026-07-12 | — | Newer, no-API-key alternative to holehe: 2,500+ platform checks + breach detection + identity clustering, pip-installable. Worth a bake-off against holehe. |
| [khast3x/h8mail](https://github.com/khast3x/h8mail) | 5,086 | 2026-07-12 | — | Email OSINT + breach-hunting; chases down related emails and password-breach exposure. Useful as a **risk score input**, not for the enrichment itself. |
| [umuterturk/email-verifier](https://github.com/umuterturk/email-verifier) | 516 | 2026-07-12 | MIT | Smaller, privacy-first Go verifier — a lighter fallback/comparison to AfterShip's. |
| [trumail/trumail](https://github.com/trumail/trumail) | 1,049 | 2026-07-08 | BSD-3 | Deployable email-verification **API service** (not just a library) — useful if you want a standalone microservice rather than an embedded library. |

## 4. Phone number OSINT

| Repo | ★ | Updated | License | Why it matters |
|---|---|---|---|---|
| [sundowndev/phoneinfoga](https://github.com/sundowndev/phoneinfoga) | 16,975 | 2026-07-12 | GPL-3.0 | The standard OSS phone-number intelligence framework: carrier, line type (mobile/VOIP/landline), country, and reputation/scam scanners. Directly gives you a "is this a real, non-VOIP/burner number" validation signal for leads. |
| [TermuxHackz/X-osint](https://github.com/TermuxHackz/X-osint) | 2,481 | 2026-07-12 | GPL-3.0 | Lighter phone/username OSINT script — useful as a secondary cross-check, not primary. |
| [megadose/phoneinfoga-maltego](https://github.com/megadose/phoneinfoga-maltego) | 131 | 2026-06-28 | GPL-3.0 | Maltego transform wrapper if you ever want a visual graph view of phone-linked entities. |

## 5. Person / social-footprint validation ("is this a real, active person?")

| Repo | ★ | Updated | License | Why it matters |
|---|---|---|---|---|
| [sherlock-project/sherlock](https://github.com/sherlock-project/sherlock) | 86,455 | 2026-07-12 | MIT | The reference username-enumeration tool across 400+ sites — highest star count and still actively updated daily. |
| [soxoj/maigret](https://github.com/soxoj/maigret) | 35,298 | 2026-07-12 | MIT | Sherlock's more advanced sibling — 3,000+ sites, builds a "dossier" per username with extracted bio/location data, HTML/PDF report export. Best default choice for the identity-validation module. |
| [qeeqbox/social-analyzer](https://github.com/qeeqbox/social-analyzer) | 23,440 | 2026-07-12 | AGPL-3.0 | API + CLI + web app for scoring how likely a profile match is real across 1,000 sites — useful for confidence scoring rather than a flat yes/no. |
| [mxrch/GHunt](https://github.com/mxrch/GHunt) | 19,206 | 2026-07-12 | — | Deep Google-account OSINT (Gmail → linked Calendar, Maps reviews, YouTube, Photos exposure) — powerful validation signal from just an email, but review ToS carefully (see Section 9). |

## 6. Company / firmographic enrichment

| Repo | ★ | Updated | License | Why it matters |
|---|---|---|---|---|
| [Lead-Orchestra/awesome-b2b-leads](https://github.com/Lead-Orchestra/awesome-b2b-leads) | 15 | 2026-07-12 | MIT-ish | Curated meta-list mapping the **entire commercial B2B enrichment stack** (Clearbit, Apollo, ZoomInfo, People Data Labs, Clay, FullContact, NeverBounce/ZeroBounce/Kickbox for email, Bright Data/Oxylabs/ScraperAPI for proxies, plus n8n workflow templates). Use this as your reference map for the paid-API layer you'll wrap around the OSS core. |
| [codyschneiderx/waterfall-gtm](https://github.com/codyschneiderx/waterfall-gtm) | 32 | 2026-06-21 | MIT | Open-source "waterfall" enrichment pattern — call provider A, fall back to B/C only on miss, to control enrichment-API cost. Directly reusable architecture for your cost-conscious enrichment pipeline. |
| [rqcai200/lead-enrichment-scoring](https://github.com/rqcai200/lead-enrichment-scoring) | 23 | 2026-07-07 | MIT | Self-hosted LinkedIn-based enrichment + scoring positioned as a cheap Clay alternative — good reference implementation. |
| [mattvinall/Quick-Enrich-Tools](https://github.com/mattvinall/Quick-Enrich-Tools) | 3 | 2026-06-05 | MIT | Six small, cloneable, self-hostable enrichment tools — good for cannibalizing specific functions rather than adopting wholesale. |
| [rahulchhabria/local-enrichment-tool](https://github.com/rahulchhabria/local-enrichment-tool) | 4 | 2026-03-24 | MIT | AI-powered domain → firmographic/technographic/hiring-data enrichment, runs locally. |

## 7. Web data-extraction infrastructure (pulling data off ad landing pages & sites)

| Repo | ★ | Updated | License | Why it matters |
|---|---|---|---|---|
| [firecrawl/firecrawl](https://github.com/firecrawl/firecrawl) | 149,878 | 2026-07-12 | AGPL-3.0 | The most-starred web-data API on GitHub: turns any URL into clean, LLM-ready structured data (markdown/JSON), handles JS rendering, crawling, and search — the fastest way to turn an ad landing page or business site into structured lead fields. |
| [browser-use/browser-use](https://github.com/browser-use/browser-use) | 104,404 | 2026-07-13 | MIT | Lets an AI agent drive a real browser to interact with a page (forms, logins, dynamic content) — directly useful for agent-driven enrichment tasks that need clicking/scrolling, not just static fetch. |
| [unclecode/crawl4ai](https://github.com/unclecode/crawl4ai) | 72,451 | 2026-07-12 | Apache-2.0 | Purpose-built "LLM-friendly" crawler — fast, async, outputs clean markdown/JSON for feeding straight into an enrichment LLM step. |
| [D4Vinci/Scrapling](https://github.com/D4Vinci/Scrapling) | 69,291 | 2026-07-12 | BSD-3 | Adaptive scraper that self-heals when a site's markup changes — reduces the maintenance burden of brittle selectors on ad-network landing pages. |
| [scrapy/scrapy](https://github.com/scrapy/scrapy) | 63,110 | 2026-07-12 | BSD-3 | The battle-tested, mature Python crawling framework — best choice if you need large-scale, scheduled, distributed crawling rather than one-off agent fetches. |
| [apify/crawlee](https://github.com/apify/crawlee) | 24,674 | 2026-07-12 | Apache-2.0 | Production-grade Node/TS crawler with built-in proxy rotation and storage — pick this over Scrapy if your stack is JS/TS. |
| [microsoft/playwright](https://github.com/microsoft/playwright) | 92,686 | 2026-07-12 | Apache-2.0 | Underlying browser-automation engine most of the above tools build on; use directly if you need custom low-level control. |

## 8. Orchestration frameworks (tie every module into one pipeline)

| Repo | ★ | Updated | License | Why it matters |
|---|---|---|---|---|
| [smicallef/spiderfoot](https://github.com/smicallef/spiderfoot) | 19,507 | 2026-07-12 | MIT | The most mature OSS OSINT **automation engine** — 200+ modules, module dependency graph, web UI, correlation engine that flags risk. Strongest candidate as the backbone orchestrator you plug your custom modules into rather than building an orchestrator from scratch. |
| [lanmaster53/recon-ng](https://github.com/lanmaster53/recon-ng) | 5,769 | 2026-07-12 | GPL-3.0 | Metasploit-style modular framework with a marketplace of modules — good architectural reference (module system, workspaces, reporting) even if you don't adopt it directly. |
| [kpcyrd/sn0int](https://github.com/kpcyrd/sn0int) | 2,479 | 2026-07-11 | GPL-3.0 | Semi-automatic OSINT framework with its own package manager for modules (Lua-scriptable) — interesting model for letting your AI agents write and register new "modules" as first-class artifacts. |
| [lockfale/OSINT-Framework](https://github.com/lockfale/OSINT-Framework) | 11,679 | 2026-07-12 | MIT | Not code — an interactive **link directory** of OSINT resources by category. Useful as a checklist when agents scope new modules ("what data sources exist for X?"). |

## 9. AI-agent-native OSINT (directly relevant to your multi-agent build plan)

| Repo | ★ | Updated | License | Why it matters |
|---|---|---|---|---|
| [OpenOSINT/OpenOSINT](https://github.com/OpenOSINT/OpenOSINT) | 967 | 2026-07-12 | MIT | AI-powered OSINT agent with an interactive REPL, MCP server, and CLI (16 tools); works with Claude/GPT-4/local models — closest existing analog to what you're building. Worth a deep-dive PR to assess reusing its MCP server layer. |
| [frishtik/osint-tools-mcp-server](https://github.com/frishtik/osint-tools-mcp-server) | 219 | 2026-07-07 | MIT | MCP server exposing multiple OSINT tools to Claude-style agents — a template for wrapping the tools above as agent-callable functions. |
| [mukul975/Anthropic-Cybersecurity-Skills](https://github.com/mukul975/Anthropic-Cybersecurity-Skills) | 25,402 | 2026-07-12 | Apache-2.0 | 817 structured cybersecurity/OSINT skills mapped to MITRE ATT&CK and other frameworks, designed for AI agents — a large prompt/skill library your agents can draw on. |
| [alphaparkinc/genpark-automated-leads-enrichment-skill](https://github.com/alphaparkinc/genpark-automated-leads-enrichment-skill) | 3 | 2026-07-12 | — | Small but directly on-target: an agent "skill" that performs corporate-attribute lookups and drafts outreach — good pattern reference even if too small to adopt outright. |

## 10. Curated meta-lists (use for gap-scanning, not for adoption)

- [jivoi/awesome-osint](https://github.com/jivoi/awesome-osint) — 27,365★ — the largest general OSINT link directory.
- [osintambition/Social-Media-OSINT-Tools-Collection](https://github.com/osintambition/Social-Media-OSINT-Tools-Collection) — 1,877★ — social-platform-specific tool index.
- [cipher387/maltego-transforms-list](https://github.com/cipher387/maltego-transforms-list) — 274★ — catalog of Maltego transforms if you ever want a visual link-analysis front end.
- [neospl0it/osint-bookmark](https://github.com/neospl0it/osint-bookmark) — 220★ — company/DNS/WHOIS-focused bookmark list, closely aligned with your "enrich a company/website" use case.

---

## 11. Compliance & risk notes (read before agents start building)

Since you have explicit permission for the ads/websites you're validating against, most of the pipeline above is low-risk. A few categories carry real legal/ethical exposure that I'd flag for every agent and check for in every PR, especially given you (and much of your likely lead base) are in the EU/GDPR zone:

- **Reverse-image "stalk your friends" tools** (e.g., EagleEye) and **personal social-media dumping tools** (Instagram follower/location scrapers, GHunt's deep Google-account exposure) go beyond validating a submitted lead — they can pull in third-party personal data the lead never gave permission for. I'd exclude these from the core validation pipeline and only allow narrowly-scoped, documented exceptions.
- **holehe / GHunt / breach-checkers** operate in a gray zone with the platforms they query (most explicitly disclaim commercial/bulk use in their license or docs) — fine for spot-checking a handful of suspicious leads, risky if run at ad-campaign scale against every lead. Rate-limit and document the legal basis (legitimate interest / anti-fraud) per GDPR Art. 6.
- **LinkedIn scrapers** (several in Section 6's neighborhood) violate LinkedIn's ToS outright and carry real legal precedent (hiQ v. LinkedIn and later rulings cut both ways) — treat as "reference only, do not deploy" unless you get explicit written legal sign-off.
- Store enrichment/validation results with a clear **data retention and deletion policy** and a log of the legal basis + consent/permission reference for each source, since you're processing personal data (email, phone, name) under GDPR as a Portugal-based operator.

I'll enforce these as hard PR-review gates in Stage 2 (see Section 14).

---

## 12. Recommended repo structure (Stage 1 → Stage 2)

```
osint-lead-platform/
├── README.md                     # Project overview, architecture diagram, stage status
├── docs/
│   ├── research/                 # This document + per-tool deep-dive notes (one .md per candidate)
│   ├── compliance.md             # GDPR/legal basis notes from Section 11, kept living
│   └── architecture.md           # Pipeline diagram, module contracts, data schema
├── modules/                      # One folder per capability, added only in Stage 2
│   ├── domain-intel/
│   ├── email-validate/
│   ├── phone-validate/
│   ├── social-footprint/
│   ├── company-enrich/
│   └── extraction/
├── evaluations/                  # Stage 1 output: one markdown scorecard per tool evaluated
│   └── TEMPLATE.md
├── .github/
│   ├── PULL_REQUEST_TEMPLATE.md
│   └── ISSUE_TEMPLATE/tool-evaluation.md
└── CONTRIBUTING.md                # How agents should scope work, branch, and open PRs
```

For Stage 1, only `docs/research/`, `evaluations/`, and the templates need to exist — no `modules/` code yet, so agents can't jump ahead of the research phase.

---

## 13. Ready-to-use prompts for delegating Stage-1 research tasks to other AI agents

Give each agent **one tool, one deliverable file**, and require the exact scorecard fields below so every PR is comparable. Template (fill `{TOOL}`, `{REPO_URL}`, `{CATEGORY}`):

> You are a research contributor on the `osint-lead-platform` repo, Stage 1 (research only — do not write integration code). Your task: produce `evaluations/{tool-slug}.md` evaluating **{TOOL}** ({REPO_URL}) for the "{CATEGORY}" module of a lead-enrichment/validation platform.
>
> Requirements for the file:
> 1. **Summary** — what it does, in 3 sentences.
> 2. **License** — exact license, and whether it permits commercial/internal-business use without restriction (flag AGPL/"no commercial use" clauses explicitly).
> 3. **Maintenance health** — last commit date, open issue count, whether it's a single-maintainer project (bus-factor risk).
> 4. **Input/output contract** — what data goes in, what structured data comes out (show a real example, don't paraphrase).
> 5. **Dependencies & runtime** — language, install method, any required API keys, expected latency for a single lookup.
> 6. **Rate limits / ToS risk** — does it hit third-party sites/APIs in a way that could violate their ToS at scale? Cite the specific docs/README section.
> 7. **Fit score (1-5)** for our specific use case (validating leads with permission) + one paragraph justification.
> 8. **Recommendation** — adopt as-is / fork & modify / reference only / reject, with reasoning.
>
> Do not modify anything outside `evaluations/{tool-slug}.md`. Open a PR titled `research: evaluate {TOOL}` against `main`. Cite every claim with a link to the exact file/line/doc you're referencing.

Batch these out one PR per tool from the priority list: `web-check`, `AfterShip/email-verifier`, `holehe`, `phoneinfoga`, `maigret`, `spiderfoot`, `firecrawl`, `crawl4ai`, `OpenOSINT`, `theHarvester`, `waterfall-gtm`, `h8mail`.

---

## 14. My PR review checklist (Stage 1)

For every `evaluations/*.md` PR, I will check, in order:

1. **Scope discipline** — touches only the one file it was assigned; flag and request changes if it adds code, deps, or touches other evaluations.
2. **Sourcing** — every factual claim (license, star count, last-commit date, rate limits) links to the actual GitHub page/doc it came from; reject unsupported claims.
3. **License accuracy** — cross-check the stated license against the repo's actual `LICENSE` file; AGPL/non-commercial licenses must be flagged loudly, not buried.
4. **Compliance flag** — confirm the agent addressed ToS/GDPR risk per Section 11 for anything touching personal data or third-party platforms.
5. **Fit score justification** — score must connect to our actual pipeline (Section 1 table), not generic praise.
6. **Actionable recommendation** — "adopt/fork/reference/reject" must be unambiguous and followed by a concrete next step if adopted.

After merge, I'll aggregate all scorecards into a single decision doc and propose which modules move to Stage 2 (code), which need a second-opinion evaluation, and which are rejected — before any implementation branch opens.

---

## Sources

All data points cited above were pulled directly from GitHub repository metadata (stars, license, last-updated date) and repository READMEs via the GitHub search/API on 2026-07-13. Direct links are included in every table row above.
