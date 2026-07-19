# Evaluation: local-enrichment-tool

- **Repo:** https://github.com/rahulchhabria/local-enrichment-tool
- **Target module:** `company-enrich` (Enrich stage — company/firmographic enrichment)
- **Evaluator:** Devin (Stage 1 second-opinion)
- **Date:** 2026-07-19

## 1. Summary

`rahulchhabria/local-enrichment-tool` is a one-day, single-author TypeScript demo that takes a company domain (or name or LinkedIn URL) and produces a Markdown report of firmographic, technographic, funding, leadership, hiring, GitHub, and mobile-app data. It uses Anthropic Claude via the Vercel AI SDK to synthesize raw scraped inputs into a structured `CompanyEnrichmentData` object. The repository is 63 KB, 4 stars, 0 forks, and has not been maintained beyond its initial 2026-02-15/16 push. It is **not adoptable as a library or subprocess** for the platform: it is Node/TypeScript (not Go/Python), requires a paid LLM key, scrapes LinkedIn public search pages, relies on LLM inference rather than deterministic sources, and outputs Markdown by default. Its main value to us is as a **field-shape reference** for the kind of company data a `company-enrich` module might expose, and as a cautionary example of what to avoid (LinkedIn scraping, LLM hallucination, Anthropic dependency).

## 2. License

- **License:** MIT License ([LICENSE](https://github.com/rahulchhabria/local-enrichment-tool/blob/main/LICENSE), `licenseInfo.key = mit` via GitHub API).
- **Commercial/internal-business use allowed without restriction?** **Yes**, for the repo's own code. MIT permits use, modification, and distribution with attribution. However, the repo's behavior calls third-party services and websites (LinkedIn, Crunchbase, GitHub, App Store, Play Store, job boards, Anthropic) whose ToS govern the actual data access. The MIT license on this code does **not** grant rights over those third-party data sources.
- **AGPL / "no commercial use" style clauses?** **None** in the repo's own license. The Anthropic API and LinkedIn/Apple/Google/job-board ToS are the real usage constraints.

## 3. Maintenance health

- **Last commit:** 2026-02-16 (`pushedAt` 2026-02-16T04:17:37Z; `createdAt` 2026-02-15T18:15:11Z). The repo was effectively built in a single day and has had no subsequent maintenance.
- **Open issues:** 0 (`hasIssuesEnabled` true, `issues.totalCount` 0). Low adoption, not proven stability.
- **Contributors:** 2 — `rahulchhabria` (6 commits) and `claude` (3 commits). The `claude` contributor is consistent with the README's "Built with Claude" claim. **Single-maintainer / AI-generated demo risk: YES.**
- **Release cadence:** None. No GitHub Releases, no tags, no npm package. Install is `git clone` + `npm install`.
- **Repository size / language:** 63 KB, 100% TypeScript.

## 4. Input / output contract

### Input

```ts
export interface CompanyEnrichmentInput {
  domain?: string;
  companyName?: string;
  linkedinUrl?: string;
}
```

From `src/types/enrichment.ts` ([source](https://github.com/rahulchhabria/local-enrichment-tool/blob/main/src/types/enrichment.ts)) and `src/lib/enrichment-engine.ts` ([source](https://github.com/rahulchhabria/local-enrichment-tool/blob/main/src/lib/enrichment-engine.ts)):

```bash
npm run enrich -- stripe.com
# or from a file
npm run enrich -- --file domains.txt
```

### Output

Default output is a Markdown file in `./output/<domain>.md`. The internal `CompanyEnrichmentData` TypeScript interface defines a rich structured shape:

```ts
export interface CompanyEnrichmentData {
  name: string;
  legalName?: string;
  domain: string;
  website: string;
  description: string;
  founded?: number;
  employeeCount?: number;
  employeeCountRange?: string;
  engineeringCount?: number;
  headquarters: { city?: string; state?: string; country?: string; address?: string; };
  industry: string[];
  vertical: string[];
  keywords: string[];
  totalFundingRaised?: string;
  latestFundingRound?: FundingRound;
  ceo?: { name: string; linkedinUrl?: string; };
  founders?: Array<{ name: string; role?: string; linkedinUrl?: string; }>;
  openPositions?: number;
  jobPostings?: JobPosting[];
  linkedinUrl?: string;
  twitterUrl?: string;
  crunchbaseUrl?: string;
  githubUrl?: string;
  githubActivity?: { ... };
  hiring?: { ... };
  technographic?: { ... };
  mobileApps?: { ... };
  aiInsights?: { ... };
}
```

This is a useful **field catalog** for `company-enrich`, but the repo does not expose it as a clean JSON CLI or library API. The Markdown-first output and TypeScript/Node runtime make it an awkward fit for a Go-based module.

## 5. Dependencies & runtime

- **Language / runtime:** Node.js + TypeScript (uses `tsx` for dev). `type: "module"`.
- **Install method:** `git clone` + `npm install` only. No npm package.
- **Required API keys / accounts:**
  - `ANTHROPIC_API_KEY` (required) — Claude via Vercel AI SDK; this is a paid service.
  - `GITHUB_TOKEN` (optional) — raises GitHub API rate limits from 60 to 5,000 req/hr.
- **Key runtime dependencies:** `@ai-sdk/anthropic`, `ai`, `axios`, `cheerio`, `express`, `zod`, `@sentry/node`.
- **Expected latency for a single lookup:** Not documented. In practice it makes sequential HTTP calls (website, LinkedIn public search, Crunchbase public page, GitHub API, job boards) and then a Claude `generateObject` call. Latency is likely 10–60+ seconds depending on network and LLM response time. No explicit timeout configuration is exposed.

## 6. Rate limits / ToS risk

This is the dominant risk area.

- **LinkedIn scraping.** `src/lib/linkedin-headcount.ts` fetches `https://www.linkedin.com/company/<company>` and `https://www.linkedin.com/search/results/people/?keywords=...` with a desktop User-Agent to estimate total and engineering headcount. The source code comments cite `hiQ Labs v. LinkedIn` as a legal basis for scraping public data, but the platform's `docs/compliance.md` explicitly treats LinkedIn scraping as **out of scope** for production. The file is currently a mock (`fetchLinkedInData` returns `''` with a console note), but the `LinkedInHeadcountFetcher` class is wired in and the author clearly intends it to be live.
- **Crunchbase public-page scraping.** `searchCrunchbase` constructs `https://www.crunchbase.com/organization/<slug>` and fetches it with a generic User-Agent. Crunchbase ToS prohibits automated scraping; this is a ToS risk and brittle (HTML-dependent).
- **Job board scraping.** `src/lib/job-scraper.ts` targets Greenhouse, Lever, and Ashby public job boards. These are generally intended for public use, but bulk/automated scraping may still hit rate limits or ToS.
- **GitHub API.** Calls are unauthenticated by default, which limits to 60 requests/hour. A token is optional.
- **Anthropic API.** Paid-only; no free path. The entire structured output is generated by Claude, so results are non-deterministic and may hallucinate.
- **LLM-generated PII.** The `CompanyEnrichmentData` interface includes `ceo`, `founders`, `leadershipChanges`, and `jobPostings` with names and LinkedIn URLs. Generating and storing these from an LLM prompt raises GDPR accuracy and legal-basis questions beyond simple public-fact enrichment.

## 7. Fit score (1-5)

**Score:** 2

**Justification:** The repository is squarely in the `company-enrich` lane and the `CompanyEnrichmentData` interface is a good reference for the fields a `company-enrich` module might return. That saves it from a 1. However, it scores only 2 because:

1. It is a one-day, single-maintainer (plus AI) demo with no releases, no npm package, and no maintenance track record.
2. It requires a **paid Anthropic API key** and uses an LLM to synthesize structured data, which conflicts with the platform's preference for deterministic, low-risk, auditable enrichment.
3. It includes **LinkedIn scraping intent** and Crunchbase page scraping, both ToS-grey and explicitly restricted by `docs/compliance.md`.
4. It is **TypeScript/Node**, so it cannot be embedded in the Go module pipeline without a separate Node runtime or HTTP wrapper.
5. It outputs Markdown files by default, not the JSON-on-stdout, audit-on-stderr contract that `modules/<name>/` follow.

For the platform's `company-enrich` module, it is **not adoptable**.

## 8. Recommendation

**Reject for adoption / reference only for field-shape and anti-patterns.**

**Reasoning + concrete next step:** Do not import, fork, or subprocess this tool. The maintenance, licensing-of-data, ToS, LLM-cost, and runtime mismatches are too severe. The `CompanyEnrichmentData` interface can inform the *shape* of the future `company-enrich` result (e.g. `name`, `legalName`, `domain`, `headquarters`, `industry`, `employeeCount`, `founded`, `description`, `linkedinUrl`, `githubUrl`, `technographic`) but the *sourcing* must come from deterministic public sources and optional paid adapters, not from Claude synthesis or LinkedIn scraping. The planned `company-enrich` module should explicitly exclude LinkedIn scraping and LLM inference of company facts.

## Sources

- Repo metadata: `rahulchhabria/local-enrichment-tool` — 4 stars, 0 forks, 1 maintainer + `claude` contributor, created 2026-02-15, last pushed 2026-02-16, 63 KB, 100% TypeScript, MIT license. GitHub API via `gh repo view`.
- README: "AI-powered company enrichment tool... runs entirely on your machine... built with Claude" ([README.md](https://github.com/rahulchhabria/local-enrichment-tool/blob/main/README.md)).
- `package.json`: dependencies include `@ai-sdk/anthropic`, `ai`, `axios`, `cheerio`, `express`, `zod`; scripts `npm run enrich` and `npm run dev` ([package.json](https://github.com/rahulchhabria/local-enrichment-tool/blob/main/package.json)).
- `.env.example`: `ANTHROPIC_API_KEY` required, `GITHUB_TOKEN` optional ([.env.example](https://github.com/rahulchhabria/local-enrichment-tool/blob/main/.env.example)).
- `src/types/enrichment.ts`: full `CompanyEnrichmentData` schema ([source](https://github.com/rahulchhabria/local-enrichment-tool/blob/main/src/types/enrichment.ts)).
- `src/lib/enrichment-engine.ts`: orchestration logic, `findDomainFromName`, `fetchLinkedInData` (mocked, returns `''`), `searchCrunchbase` (fetches Crunchbase public pages), GitHub/org/job-board/tech/mobile/social extraction, final `extractWithAI` call to Claude ([source](https://github.com/rahulchhabria/local-enrichment-tool/blob/main/src/lib/enrichment-engine.ts)).
- `src/lib/linkedin-headcount.ts`: LinkedIn public company page and people-search scraping to estimate total/engineering headcount ([source](https://github.com/rahulchhabria/local-enrichment-tool/blob/main/src/lib/linkedin-headcount.ts)).
- Project compliance constraints: `docs/compliance.md` (LinkedIn scraping excluded from production; breach/personal-surveillance tools restricted; GDPR Art. 6 logging required).
