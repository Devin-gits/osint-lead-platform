# Evaluation: web-check

- **Repo:** https://github.com/lissy93/web-check
- **Target module:** `domain-intel` (Ingest stage)
- **Evaluator:** Claude (AI research contributor)
- **Date:** 2026-07-13

## 1. Summary

web-check is an all-in-one, self-hostable website/domain OSINT dashboard that runs ~35 independent checks against a single target URL — DNS records, SSL/TLS, HTTP headers & security, WHOIS, tech stack, open ports, mail config (SPF/DKIM/DMARC), blocklist/threat status, social tags, screenshots and more ([README "Web-Check" intro & feature list](https://github.com/lissy93/web-check/blob/master/README.md)). Each check is exposed both as a React GUI card and as a standalone JSON API endpoint (`/api/<check>?url=<target>`), so it can be consumed programmatically ([`api/` directory](https://github.com/lissy93/web-check/tree/master/api)). For the `domain-intel` module it is a strong fit as the "validate this lead's website" first pass, since it turns a bare domain into a broad structured profile with no API keys required by default ([README "Configuring": "By default, no configuration is needed"](https://github.com/lissy93/web-check/blob/master/README.md)).

## 2. License

- **License:** MIT ([`LICENSE` file](https://github.com/lissy93/web-check/blob/master/LICENSE), SPDX `MIT` per [GitHub license API](https://api.github.com/repos/lissy93/web-check); README restates it: ["licensed under MIT © Alicia Sykes 2023 - 2026"](https://github.com/lissy93/web-check/blob/master/README.md)).
- **Commercial/internal-business use allowed without restriction?** yes — the MIT grant explicitly permits use, copy, modify, merge, publish, distribute, sublicense "and/or sell" copies, with the only condition being retention of the copyright/permission notice ([`LICENSE`](https://github.com/lissy93/web-check/blob/master/LICENSE)).
- **AGPL / "no commercial use" style clauses?** None present. The `LICENSE` file is stock MIT with **no copyleft, no network-use (AGPL §13) clause, and no non-commercial restriction** ([`LICENSE`](https://github.com/lissy93/web-check/blob/master/LICENSE)). Note the README asks for optional GitHub sponsorship to cover public-instance costs, but this is a request, **not** a license term ([README "Sponsor" / "Supporting" section](https://github.com/lissy93/web-check/blob/master/README.md)). Transitive dependency licenses are tracked separately via the repo's FOSSA SBOM badge and should be reviewed before shipping ([FOSSA report link in README](https://github.com/lissy93/web-check/blob/master/README.md)).

## 3. Maintenance health

- **Last commit:** 2026-07-11 — commit "🔖 Bump version to 2.1.10" on `master` ([commits API for `master`](https://api.github.com/repos/lissy93/web-check/commits/master); repo `pushed_at` = `2026-07-11T07:28:44Z` per [repo API](https://api.github.com/repos/lissy93/web-check)).
- **Open issues:** 57 open issues (excluding PRs) ([GitHub issue search `type:issue state:open`](https://github.com/lissy93/web-check/issues?q=is%3Aissue+is%3Aopen)); plus 23 open PRs ([PR search](https://github.com/lissy93/web-check/pulls)). The repo-level `open_issues_count` of 80 counts both together.
- **Contributors:** 33 ([contributors API, last page = 33](https://api.github.com/repos/lissy93/web-check/contributors?per_page=1); [contributors graph](https://github.com/lissy93/web-check/graphs/contributors)). This is a **maintainer-led** project — Alicia Sykes (lissy93) is the dominant author and sole release-tagger — so bus-factor risk is real (**yes, moderate**): breadth of contributors exists but direction and releases depend on one person.
- **Release cadence:** Active and recent. Currently at v2.1.10 with a version bump landing 2026-07-11, and 34,130 stars indicating a well-exercised, popular project ([repo API `stargazers_count`](https://api.github.com/repos/lissy93/web-check); [releases](https://github.com/lissy93/web-check/releases)).

## 4. Input / output contract

**Contract:** HTTP `GET /api/<check-name>?url=<target>` → `200` with a JSON body specific to that check (`Content-Type: application/json`); errors return `{ "error": "..." }` with a 4xx/5xx status ([`api/_common/middleware.js`](https://github.com/lissy93/web-check/blob/master/api/_common/middleware.js)). Below are **real outputs**, produced by running the actual handler code from a fresh clone (commit v2.1.10) in NODE mode against live targets.

```
# input — DNS check (api/dns.js) against example.com
GET /api/dns?url=https://example.com

# output (verbatim, trimmed to representative fields)
HTTP 200
{
  "A": ["172.66.147.243", "104.20.23.154"],
  "AAAA": ["2606:4700:10::ac42:93f3", "2606:4700:10::6814:179a"],
  "MX": [{ "exchange": "", "priority": 0 }],
  "TXT": [["_k2n1y4vw3qtb4skdx9e7dxt97qrmmq9"], ["v=spf1 -all"]],
  "NS": ["hera.ns.cloudflare.com", "elliott.ns.cloudflare.com"],
  "CNAME": [],
  "SOA": {
    "nsname": "elliott.ns.cloudflare.com",
    "hostmaster": "dns.cloudflare.com",
    "serial": 2407636105, "refresh": 10000, "retry": 2400,
    "expire": 604800, "minttl": 1800
  },
  "SRV": [],
  "PTR": []
}

# input — IP check (api/get-ip.js) against github.com
GET /api/get-ip?url=https://github.com

# output (verbatim)
HTTP 200
{ "ip": "140.82.113.4", "family": 4 }
```

Source handlers: [`api/dns.js`](https://github.com/lissy93/web-check/blob/master/api/dns.js), [`api/get-ip.js`](https://github.com/lissy93/web-check/blob/master/api/get-ip.js), [`api/_common/parse-target.js`](https://github.com/lissy93/web-check/blob/master/api/_common/parse-target.js).

## 5. Dependencies & runtime

- **Language / runtime:** Node.js ≥ 18.16.1, JavaScript/TypeScript (React front end + Node serverless-style API handlers) ([README "Developing"](https://github.com/lissy93/web-check/blob/master/README.md)). Some checks additionally need `chromium`, `traceroute`, and `dns` binaries present, and are silently skipped if absent ([README "Developing"](https://github.com/lissy93/web-check/blob/master/README.md)).
- **Install method:** Docker (`docker run -p 3000:3000 lissy93/web-check`, image on [DockerHub](https://hub.docker.com/r/lissy93/web-check) / GHCR); 1-click Netlify or Vercel deploy; or from source (`git clone` → `yarn install` → `yarn build` → `yarn serve`) ([README "Deployment"](https://github.com/lissy93/web-check/blob/master/README.md)).
- **Required API keys / accounts:** **None required** — "By default, no configuration is needed." Optional keys unlock or raise limits on specific checks: Shodan, Google Cloud, WhoAPI, SecurityTrails, Cloudmersive, Tranco, URLScan, BuiltWith, and a torrent API ([README "Configuring" API-keys table](https://github.com/lissy93/web-check/blob/master/README.md)).
- **Expected latency for a single lookup:** Sub-second for local checks — the `api/dns.js` handler returned in **~9 ms** and `api/get-ip.js` near-instantly in my local runs (warm resolver cache). A full multi-check scan aggregates ~35 checks and takes seconds; the public instance caps each request via `PUBLIC_API_TIMEOUT_LIMIT` (default 40000 ms in [`api/_common/middleware.js`](https://github.com/lissy93/web-check/blob/master/api/_common/middleware.js)), and notes the public instance uses a lower limit to control cost.

## 6. Rate limits / ToS risk

Risk is **low for the default/core checks** but **conditional for the optional external-API checks**. The DNS, IP, SSL/TLS, headers, and WHOIS-style checks query the target domain and public infrastructure directly (standard for validating a business's own website — matching the "Low" rating for domain intel in [`docs/compliance.md` risk table](docs/compliance.md)). However, several checks call third-party services that each impose their own ToS and rate limits: the README states optional API keys exist "to increase rate-limits for some checks that use external APIs" ([README "Configuring"](https://github.com/lissy93/web-check/blob/master/README.md)), and the blocklist/threat checks hit sources such as abuse.ch's [URLHaus](https://github.com/lissy93/web-check/blob/master/README.md) plus Shodan, Google, URLScan, BuiltWith, Tranco, and SecurityTrails via the keyed integrations ([README API-keys table](https://github.com/lissy93/web-check/blob/master/README.md)). Running the full scan against **every** lead at ad-campaign scale would multiply calls to those upstreams and risk tripping their per-key/free-tier limits or ToS. Mitigations built in: self-hosting removes the shared public-instance cap, and `API_ENABLE_RATE_LIMIT` lets you rate-limit your own `/api` endpoints ([README config table](https://github.com/lissy93/web-check/blob/master/README.md)). Recommended: run only the keyless domain-infra checks by default and gate the keyed/external checks behind selective, rate-limited use per [`docs/compliance.md` hard rule #3](docs/compliance.md).

## 7. Fit score (1-5)

**Score:** 5

**Justification** (connected to the `README.md` pipeline table):

Our pipeline table assigns `modules/domain-intel` to the **Ingest** stage with web-check listed as a candidate ([repo `README.md` pipeline table](README.md)). web-check maps almost exactly onto that stage's job: take the raw lead's `domain` field and produce a broad, structured profile (DNS, SSL, headers, WHOIS, tech stack, mail config, blocklist status) that downstream Enrich/Validate stages consume — e.g., domain-age/WHOIS and blocklist signals feed the "is this a real, low-risk business?" validation gate. It scores a 5 rather than lower because (a) the default checks need **no API keys and no paid accounts**, matching a cost-conscious OSS core; (b) it is **MIT-licensed**, so we can fork and embed it in a commercial platform without copyleft friction (unlike several GPL/AGPL candidates in the research doc); (c) it already exposes a **clean per-check JSON API**, so we can call individual endpoints (e.g. `dns`, `ssl`, `whois`) as module functions instead of scraping a UI; and (d) domain intel is the **lowest personal-data-risk** category in [`docs/compliance.md`](docs/compliance.md). The one point of friction (external-API ToS/limits) is confined to optional checks we can disable, so it does not lower the core fit.

## 8. Recommendation

**Fork & modify**

**Reasoning + concrete next step:** Adopt web-check as the basis for `modules/domain-intel`, but **fork rather than adopt-as-is**. We don't need the full React GUI or all 35 checks — we need a headless subset (DNS, IP, SSL/TLS, headers, HTTP-security, WHOIS/rank, mail-config, blocklist/threats) callable as library/API functions and wired into the pipeline's Ingest stage. MIT licensing makes forking clean; we just retain the copyright notice ([`LICENSE`](https://github.com/lissy93/web-check/blob/master/LICENSE)). Concrete next step for Stage 2: open a `modules/domain-intel/` implementation PR that (1) vendors or depends on the specific `api/*.js` handlers we need with `DISABLE_GUI=true`, (2) runs keyless checks by default and puts keyed external-API checks (Shodan/URLScan/BuiltWith/etc.) behind an opt-in, rate-limited config per [`docs/compliance.md` rule #3](docs/compliance.md), (3) adds an entry to the `docs/compliance.md` risk table confirming the "Low" rating for the keyless path, and (4) reviews the FOSSA/SBOM dependency-license report before pinning versions ([README FOSSA badge](https://github.com/lissy93/web-check/blob/master/README.md)).

## Sources

- Repository metadata (license, stars, `pushed_at`, open-issue count, default branch): [GitHub repo API — lissy93/web-check](https://api.github.com/repos/lissy93/web-check)
- Latest commit date/message: [commits API — `master`](https://api.github.com/repos/lissy93/web-check/commits/master)
- Contributor count (33): [contributors API pagination](https://api.github.com/repos/lissy93/web-check/contributors?per_page=1) / [contributors graph](https://github.com/lissy93/web-check/graphs/contributors)
- Open issues vs PRs split: [issue search](https://github.com/lissy93/web-check/issues?q=is%3Aissue+is%3Aopen) / [PR list](https://github.com/lissy93/web-check/pulls)
- License terms: [`LICENSE` file](https://github.com/lissy93/web-check/blob/master/LICENSE); README license section: [README.md](https://github.com/lissy93/web-check/blob/master/README.md)
- Install / deployment / config / API keys / dev prerequisites: [README.md](https://github.com/lissy93/web-check/blob/master/README.md)
- API request/response behavior & timeout: [`api/_common/middleware.js`](https://github.com/lissy93/web-check/blob/master/api/_common/middleware.js)
- Handlers used for the real I/O examples: [`api/dns.js`](https://github.com/lissy93/web-check/blob/master/api/dns.js), [`api/get-ip.js`](https://github.com/lissy93/web-check/blob/master/api/get-ip.js), [`api/_common/parse-target.js`](https://github.com/lissy93/web-check/blob/master/api/_common/parse-target.js) — outputs generated locally from a fresh clone at v2.1.10 on 2026-07-13
- Internal alignment: [`README.md` pipeline table](README.md), [`docs/compliance.md`](docs/compliance.md)
