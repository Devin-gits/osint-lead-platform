# Evaluation: PhoneInfoga

- **Repo:** https://github.com/sundowndev/phoneinfoga
- **Target module:** `phone-validate`
- **Evaluator:** Claude (Stage 1 research agent)
- **Date:** 2026-07-13

## 1. Summary

PhoneInfoga is a Go-based framework for OSINT investigation of international phone numbers: it parses a number locally (country, format, E164) and then runs a collection of pluggable "scanners" that query external services to add carrier/line-type data, footprint search links, and reputation/disposable-number signals ([README "About"](https://github.com/sundowndev/phoneinfoga/blob/master/README.md)). It ships as a CLI, a REST API, and a browser GUI, and is usable as a Go module ([README "Features"](https://github.com/sundowndev/phoneinfoga/blob/master/README.md)). It explicitly does not verify or "track" a number and only surfaces leads for a human to investigate ([README "Anti-features"](https://github.com/sundowndev/phoneinfoga/blob/master/README.md)).

## 2. License

- **License:** GNU General Public License v3.0 (GPL-3.0) — confirmed against the repo's actual [`LICENSE` file](https://github.com/sundowndev/phoneinfoga/blob/master/LICENSE) ("GNU GENERAL PUBLIC LICENSE / Version 3, 29 June 2007") and the [GitHub license metadata](https://github.com/sundowndev/phoneinfoga) (`"key":"gpl-3.0"`).
- **Commercial/internal-business use allowed without restriction?** **Conditional.** GPL-3.0 permits commercial and internal use, but it is a **strong copyleft** license: if we link the PhoneInfoga Go module into our own code and *distribute* the result, that derivative work must itself be released under GPL-3.0. Running it internally (SaaS/back-end, no binary distribution to third parties) does **not** trigger the copyleft obligation.
- **AGPL / "no commercial use" style clauses?** **No AGPL, no "non-commercial" clause.** This is important and favorable: it is GPL-3.0, **not** AGPL-3.0, so there is **no network-use clause** — invoking PhoneInfoga as a separate process (its bundled [REST API](https://petstore.swagger.io/?url=https://raw.githubusercontent.com/sundowndev/phoneinfoga/master/web/docs/swagger.yaml) or CLI) from our MIT-licensed platform does not, by itself, force our platform code under GPL. Contrast with the AGPL-3.0 tools flagged in the [research doc](../docs/research/osint-tooling-research.md) (social-analyzer, firecrawl). License text: [LICENSE](https://github.com/sundowndev/phoneinfoga/blob/master/LICENSE).

## 3. Maintenance health

- **Last commit:** 2026-01-06 (commit `041f34a`, "docs: README.md — Add project status update") — via [commits API](https://github.com/sundowndev/phoneinfoga/commits/master). Note that commit is a **docs-only** change; the last functional release is older (see cadence).
- **Open issues:** 88 open ([GitHub repo API](https://github.com/sundowndev/phoneinfoga/issues), `open_issues_count`).
- **Contributors:** 18 ([contributors API](https://github.com/sundowndev/phoneinfoga/graphs/contributors)). Project is driven primarily by the sole owner @sundowndev, so effective **single-maintainer / bus-factor risk = yes**, compounded by the status below.
- **Release cadence:** **Effectively frozen.** Latest release is [v2.11.0, published 2024-02-21](https://github.com/sundowndev/phoneinfoga/releases) — no new release in ~2.5 years as of this evaluation. Historically releases were roughly monthly through 2023. **Critically, the README now states the project is unmaintained:** *"This project is stable but unmaintained. Upcoming bugs won't be fixed and repository could be archived at any time."* ([README "Current status"](https://github.com/sundowndev/phoneinfoga/blob/master/README.md)). This is the single most important maintenance signal for us.

## 4. Input / output contract

Real run, using the prebuilt `v2.11.0` Linux binary from the [release page](https://github.com/sundowndev/phoneinfoga/releases/tag/v2.11.0), local scanner only (no API keys configured):

```
# input
$ ./phoneinfoga scan -n "+14152007986" -D googlesearch -D numverify -D ovh

# output
Running scan for phone number +14152007986...

Results for local
Raw local: 4152007986
Local: (415) 200-7986
E164: +14152007986
International: 14152007986
Country: US

1 scanner(s) succeeded
```

With scanners enabled but no API keys, the `googlesearch` scanner additionally emits a set of **pre-built Google dork URLs** (not results) for social media and disposable-SMS sites that a human must open manually — real excerpt from the same binary:

```
Results for googlesearch
Social media:
	URL: https://www.google.com/search?q=site%3Afacebook.com+intext%3A%2215554441212%22...
	URL: https://www.google.com/search?q=site%3Alinkedin.com+intext%3A%2215554441212%22...
Disposable providers:
	URL: https://www.google.com/search?q=site%3Areceive-sms-now.com+intext%3A%2215554441212%22...
```

**Contract summary:** input is a single phone number (E164 or international format, `-n`). Output is human-readable text (or JSON via the REST API / Go module). **Key finding:** the free/offline `local` scanner returns only *country + normalized formats* — **carrier and line type require the `numverify` scanner, which needs an APILayer API key** ([scanners doc](https://github.com/sundowndev/phoneinfoga/blob/master/docs/getting-started/scanners.md)). The README's headline "carrier and line type" claim is not available offline.

## 5. Dependencies & runtime

- **Language / runtime:** Go (module requires **go 1.20**, per [`go.mod`](https://github.com/sundowndev/phoneinfoga/blob/master/go.mod)). Distributed as a **single static binary** — no runtime interpreter needed to *run* it.
- **Install method:** prebuilt binary (Linux/macOS/Windows), install script, Homebrew (`brew install phoneinfoga`), Docker (`docker pull sundowndev/phoneinfoga:latest`), or build from source / `go install` ([install doc](https://github.com/sundowndev/phoneinfoga/blob/master/docs/getting-started/install.md)). Verified locally: downloaded `phoneinfoga_Linux_x86_64.tar.gz` from v2.11.0 and ran `./phoneinfoga version` → `PhoneInfoga 2.11.0-5f6156f`.
- **Required API keys / accounts:** **none for `local` and `googlesearch`/`ovh`**; **`numverify` requires an APILayer key** and **`googlecse` requires a Google Custom Search API token + Search Engine ID** ([scanners doc](https://github.com/sundowndev/phoneinfoga/blob/master/docs/getting-started/scanners.md)). So carrier/line-type enrichment is gated behind a paid/free-tier third-party key.
- **Expected latency for a single lookup:** `local` scanner is effectively instantaneous (offline string parsing, sub-second in the run above). Scanners that hit external APIs (numverify, googlecse) add network round-trip latency and are bounded by those providers' quotas (see §6). No official per-lookup latency figure is published.

## 6. Rate limits / ToS risk

PhoneInfoga's own `local` scanner hits no third party and carries no rate-limit/ToS risk. Risk is entirely a function of **which third-party scanner modules are enabled** ([scanners doc](https://github.com/sundowndev/phoneinfoga/blob/master/docs/getting-started/scanners.md)):

- **numverify (APILayer):** requires an API key; docs state *"You can use a free API key as long as you don't exceed the monthly quota."* At lead-campaign scale we would exhaust the free tier and pay APILayer per their terms.
- **googlecse (Google Custom Search JSON API):** docs state *"100 search queries (~50 scans) per day for free... Additional requests cost $5 per 1000 queries (~500 scans), up to 10k queries per day"* — a hard per-day ceiling that caps throughput.
- **googlesearch:** generates Google search **URLs** for a human to click; it does not scrape Google programmatically, so it sidesteps automated-query ToS issues but delivers no structured data on its own.
- **ovh:** queries OVH Telecom's public API for numbers OVH owns; no key, no stated limit.

**Compliance alignment:** `docs/compliance.md` classifies Phone OSINT (PhoneInfoga) as **Low-Medium** risk, noting *"Carrier/line-type lookups are fairly standard fraud-prevention practice; scam-score lookups may hit rate limits on third-party sources"* ([compliance.md](../docs/compliance.md), per-category table). That is consistent with the findings here: the carrier/line-type signal is the low-risk, high-value part and depends on numverify's quota; the "footprint/reputation" scanners are the rate-limited, gray-zone part. Any deployment must respect the review-gate requirement of a rate-limit/usage-scope note ([compliance.md](../docs/compliance.md), "Review gate").

## 7. Fit score (1-5)

**Score:** 3

**Justification:** In the [README pipeline table](../README.md), `modules/phone-validate` sits in the **Validate** stage — its job is answering "is this a real, non-VOIP/burner number?" PhoneInfoga is the best-known OSS tool for exactly this and maps cleanly onto the stage: the `local` scanner gives us free, offline, zero-ToS-risk number validity + country normalization, and `numverify` adds the carrier/line-type signal that is the actual fraud-relevant output for lead validation. It's a single static Go binary with a REST API, which fits a modular pipeline that shells out to per-capability services, and its GPL-3.0 (non-AGPL) license does not contaminate our MIT platform when run as a separate process. **However**, the score is capped at 3, not higher, for three concrete reasons: (1) the project is **officially unmaintained and may be archived at any time** ([README](https://github.com/sundowndev/phoneinfoga/blob/master/README.md)) — a hard risk for a production dependency; (2) the headline carrier/line-type validation signal is **not available offline** and requires a third-party APILayer key with a monthly quota, so PhoneInfoga is really a thin orchestration wrapper around numverify for our use case; and (3) much of its value (footprint dorks, reputation scanners) is investigation tooling for a human analyst, not the automated yes/no validation signal our pipeline needs.

## 8. Recommendation

**Fork & modify** (leaning toward *reference / thin-adopt*).

**Reasoning + concrete next step:** Do **not** adopt as-is as a live upstream dependency, because the maintainer has declared it unmaintained and warned the repo "could be archived at any time" ([README "Current status"](https://github.com/sundowndev/phoneinfoga/blob/master/README.md)) — pinning a production validator to an abandoned repo is unacceptable. The `local`-scanner logic (E164 parsing, country/format normalization) is genuinely useful, small, and permissively usable under GPL-3.0 as a separate service. **Concrete next step for Stage 2:** stand up `modules/phone-validate` that (a) runs a **pinned fork of PhoneInfoga v2.11.0** (or its underlying Go `libphonenumber`-style parsing) as a containerized microservice invoked over its REST API — keeping the GPL boundary clean from our MIT code — and (b) calls **numverify/APILayer directly** for the carrier + line-type signal rather than depending on PhoneInfoga's abandoned scanner plumbing for it. Add the required `docs/compliance.md` rate-limit note for numverify before merge, and evaluate whether a maintained phone-parsing library (Google's `libphonenumber`) can replace the `local` scanner entirely, reducing our exposure to the archived repo.

## Sources

- Repo, description, license key, star count: https://github.com/sundowndev/phoneinfoga (GitHub repo metadata via API)
- Actual license text (GPL-3.0 v3): https://github.com/sundowndev/phoneinfoga/blob/master/LICENSE
- README (About, Current status "unmaintained", Features, Anti-features, License): https://github.com/sundowndev/phoneinfoga/blob/master/README.md
- Last commit `041f34a` (2026-01-06): https://github.com/sundowndev/phoneinfoga/commits/master
- Open-issue count (88) and contributor count (18): https://github.com/sundowndev/phoneinfoga/issues and https://github.com/sundowndev/phoneinfoga/graphs/contributors
- Releases (latest v2.11.0, 2024-02-21): https://github.com/sundowndev/phoneinfoga/releases
- Scanners, API-key requirements, and rate-limit quotes (numverify, googlecse, googlesearch, ovh, local): https://github.com/sundowndev/phoneinfoga/blob/master/docs/getting-started/scanners.md
- Install methods and Go 1.20 requirement: https://github.com/sundowndev/phoneinfoga/blob/master/docs/getting-started/install.md and https://github.com/sundowndev/phoneinfoga/blob/master/go.mod
- Real input/output: local run of the official `phoneinfoga_Linux_x86_64` v2.11.0 binary from https://github.com/sundowndev/phoneinfoga/releases/tag/v2.11.0
- REST API spec: https://petstore.swagger.io/?url=https://raw.githubusercontent.com/sundowndev/phoneinfoga/master/web/docs/swagger.yaml
- Repo pipeline table & module mapping: [README.md](../README.md); phone-OSINT risk classification: [docs/compliance.md](../docs/compliance.md); tool survey context: [docs/research/osint-tooling-research.md](../docs/research/osint-tooling-research.md)
