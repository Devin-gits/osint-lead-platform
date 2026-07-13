# Evaluation: SpiderFoot

- **Repo:** [smicallef/spiderfoot](https://github.com/smicallef/spiderfoot)
- **Target module:** Orchestration (the "tie every module into one pipeline" backbone — [research doc §8](https://github.com/Shinydev09/osint-lead-platform/blob/master/docs/research/osint-tooling-research.md))
- **Evaluator:** Claude (Stage 1 research contributor)
- **Date:** 2026-07-13

## 1. Summary

SpiderFoot is a mature (started 2012) open-source OSINT **automation engine** that ties 200+ data-source modules together in a publisher/subscriber event model, driven from a web UI or CLI and backed by SQLite. You give it a single seed entity (domain, IP, email, phone, username, person's name, etc.) and it recursively fans out across every module that can consume the events produced so far, then runs a YAML correlation engine over the results. It is built for security reconnaissance / attack-surface mapping, not for record-oriented lead enrichment. ([README](https://github.com/smicallef/spiderfoot/blob/master/README.md))

## 2. License

- **License:** MIT ([LICENSE file](https://github.com/smicallef/spiderfoot/blob/master/LICENSE); [API-reported SPDX: MIT](https://api.github.com/repos/smicallef/spiderfoot)).
- **Commercial/internal-business use allowed without restriction?** **Yes.** The MIT text grants rights "to use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies … without restriction," so embedding or forking it inside a commercial lead platform is permitted, subject only to retaining the copyright/permission notice. ([LICENSE file](https://github.com/smicallef/spiderfoot/blob/master/LICENSE))
- **AGPL / "no commercial use" style clauses?** **None.** No copyleft or non-commercial restriction in the license — this is one of the more permissive orchestrators in [research doc §8](https://github.com/Shinydev09/osint-lead-platform/blob/master/docs/research/osint-tooling-research.md) (Recon-ng and sn0int are GPL-3.0). Note the OSS project is deliberately feature-limited versus the closed commercial "SpiderFoot HX" product ([open source vs HX](https://www.spiderfoot.net/open-source-vs-hx/)), but that is a product-tier split, not a license restriction on the OSS code.

## 3. Maintenance health

- **Last commit (master):** 2023-11-05 — the newest commit on the default branch is `sfwebui: Update jQuery 3.6.0 to 3.7.1 (#1822)`. ([master commit history](https://github.com/smicallef/spiderfoot/commits/master)). The repo's `pushed_at` of 2026-04-13 reflected only a Dependabot branch (`dependabot/pip/test/pytest-9.0.3`), not merged work on master. ([branches](https://github.com/smicallef/spiderfoot/branches))
- **Latest release:** v4.0, published **2022-04-07** — no tagged release in over four years. ([releases](https://github.com/smicallef/spiderfoot/releases)).
- **Open issues:** 228 open issues, plus 39 open pull requests ([open issues](https://github.com/smicallef/spiderfoot/issues), [open PRs](https://github.com/smicallef/spiderfoot/pulls)); GitHub's combined `open_issues_count` is 268. ([API](https://api.github.com/repos/smicallef/spiderfoot))
- **Contributors:** ~52 total, but the project is effectively **single-maintainer** (Steve Micallef, whose commercial SpiderFoot HX product superseded active OSS development). Bus-factor risk: **high** — the OSS backbone is de facto in maintenance-freeze. ([contributors](https://github.com/smicallef/spiderfoot/graphs/contributors), [copyright holder in LICENSE](https://github.com/smicallef/spiderfoot/blob/master/LICENSE))
- **Release cadence:** Was steady 2012–2022 (v3.x → v4.0), then stalled; master itself has had no functional commits since late 2023. A concrete symptom: the pinned `lxml` in [`requirements.txt`](https://github.com/smicallef/spiderfoot/blob/master/requirements.txt) fails to build on current Python (3.14), requiring an unpinned install to run — see §5.

## 4. Input / output contract

SpiderFoot's module system is a **publisher/subscriber event graph**, not a linear function pipeline. Each of the 230 installed modules (`sfp_*`) is a `SpiderFootPlugin` subclass that declares `watchedEvents()` (event types it consumes), `producedEvents()` (event types it emits), and `handleEvent()` (its logic). A module fires only when another module (or the seed target) produces an event type it watches, so the graph self-assembles from the seed entity outward. ([`sfp_dnsresolve.py` L187/L202/L209](https://github.com/smicallef/spiderfoot/blob/master/modules/sfp_dnsresolve.py); [modules directory](https://github.com/smicallef/spiderfoot/tree/master/modules)). Output is a flat stream of `(source, module, type, data)` event tuples exportable as CSV/JSON/GEXF. ([README FEATURES](https://github.com/smicallef/spiderfoot/blob/master/README.md))

Real example — installed v4.0.0 locally and ran a scoped, DNS-only scan (only `sfp_dnsresolve` + `sfp_dnsraw` enabled, no third-party APIs) against `example.com`:

```
# input (CLI)
$ python3 ./sf.py -s example.com -m sfp_dnsresolve,sfp_dnsraw -o csv -q

# output (CSV: Source, Type, Data)
Source,Type,Data
SpiderFoot UI,Internet Name,example.com,example.com
SpiderFoot UI,Domain Name,example.com,example.com
sfp_dnsraw,Raw DNS Records,example.com,example.com. 35 IN MX 0 .
sfp_dnsraw,Name Server (DNS NS Records),example.com,elliott.ns.cloudflare.com
sfp_dnsraw,DNS TXT Record,example.com,v=spf1 -all
sfp_dnsraw,DNS SPF Record,example.com,v=spf1 -all
sfp_dnsresolve,Affiliate - Domain Name,elliott.ns.cloudflare.com,cloudflare.com
```

```
# same scan, JSON output (-o json) — note the event-tuple shape, no lead-record schema
[
  { "generated": 1783903072, "type": "Internet Name", "data": "example.com",
    "module": "SpiderFoot UI", "source": "example.com" },
  { "generated": 1783903072, "type": "IPv6 Address", "data": "2606:4700:10::6814:179a",
    "module": "sfp_dnsresolve", "source": "example.com" }
]
```

The key mismatch with our platform: SpiderFoot emits a **de-normalized event graph keyed by entity/type**, whereas the [architecture module contract](https://github.com/Shinydev09/osint-lead-platform/blob/master/docs/architecture.md) expects each module to take a partial lead record (JSON) and return that same record with new fields added under a namespaced key (e.g. `email_validate: {status, deliverable, …}`) without overwriting raw fields. Adapting SpiderFoot output into per-lead records would require a non-trivial reshaping/join layer.

## 5. Dependencies & runtime

- **Language / runtime:** Python 3.7+ (README states 3.7+; embeds a CherryPy web server + SQLite backend). ([README INSTALLING](https://github.com/smicallef/spiderfoot/blob/master/README.md))
- **Install method:** download the v4.0 tarball (or `git clone`), then `pip3 install -r requirements.txt`, then `python3 ./sf.py -l 127.0.0.1:5001` for the web UI or `python3 ./sf.py -s <target>` for CLI. A Dockerfile is also provided. ([README INSTALLING](https://github.com/smicallef/spiderfoot/blob/master/README.md); [requirements.txt](https://github.com/smicallef/spiderfoot/blob/master/requirements.txt)). **Caveat observed during this eval:** on Python 3.14 the pinned `lxml` in `requirements.txt` failed to build ("make sure the libxml2 and libxslt development packages are installed"); installing an unpinned `lxml` + the remaining deps was required to launch — a direct consequence of the stalled maintenance noted in §3.
- **Required API keys / accounts:** **None to run.** Most of the 200+ modules work without keys, and many that use keys have a free tier; keys are optional and only unlock specific data sources. ([README MODULES](https://github.com/smicallef/spiderfoot/blob/master/README.md))
- **Expected latency for a single lookup:** Not a "single lookup" tool — it runs a recursive multi-module scan that can range from seconds (a couple of passive modules, as in §4) to many minutes/hours for a full-footprint scan across all modules. There is no documented per-lookup latency figure; the `-u passive` / `-x` strict flags and `-max-threads` exist specifically to bound scan scope/time. ([sf.py `-h` output](https://github.com/smicallef/spiderfoot/blob/master/sf.py))

## 6. Rate limits / ToS risk

**High at scale.** SpiderFoot's value is breadth: a default scan sprays a target across dozens of third-party services (Bing, Google CSE, SHODAN, HaveIBeenPwned, VirusTotal, Hunter.io, breach/paste databases, blacklists, social-media enumeration, S3/Azure bucket probing, port scanning, etc. — see the full [MODULES table](https://github.com/smicallef/spiderfoot/blob/master/README.md)). Running this against **every inbound lead** would (a) blow through free-tier quotas on the tiered/commercial-API modules, and (b) trigger exactly the categories our own [`docs/compliance.md`](https://github.com/Shinydev09/osint-lead-platform/blob/master/docs/compliance.md) and [research doc §11](https://github.com/Shinydev09/osint-lead-platform/blob/master/docs/research/osint-tooling-research.md) flag as GDPR/ToS-risky: bulk breach-checking, non-consensual social-footprint enumeration, and pulling third-party personal data the lead never consented to. Its offensive-recon features (port scanning, subdomain-takeover checks, bucket scraping, Nmap/Nuclei tool integrations per the README) are also out of scope for validating a consented lead and could themselves breach the target's ToS. Mitigation is possible via strict module allow-listing (`-m`/`-x`/`-u passive`), but that reduces SpiderFoot to a thin wrapper over a handful of modules we could call directly.

## 7. Fit score (1-5)

**Score:** 2

**Justification** (connected to the [README pipeline table](https://github.com/Shinydev09/osint-lead-platform/blob/master/README.md) and the orchestration open question in [`docs/architecture.md`](https://github.com/Shinydev09/osint-lead-platform/blob/master/docs/architecture.md)):

SpiderFoot is a best-in-class **security-reconnaissance** engine, but it is a poor fit as the *backbone orchestrator* for this lead platform. Three mismatches drive the low score. **(1) Data model:** our [module contract](https://github.com/Shinydev09/osint-lead-platform/blob/master/docs/architecture.md) is record-in → same-record-with-namespaced-fields-out, degrading unknown fields gracefully and logging a per-call legal-basis tag; SpiderFoot instead produces an entity/type event graph (§4) with no notion of a lead record, per-field provenance in our schema, or graceful per-field `unknown` marking, so we'd write and maintain a substantial adapter either way. **(2) Scope inversion:** the platform's pipeline stages (Ingest → Enrich → Validate, per the README table) are a small, cost-controlled set of *targeted* validators (email deliverability, phone type, domain age, social footprint). SpiderFoot's design goal is the opposite — maximal recursive fan-out — which is precisely what the compliance notes (§6) tell us to avoid at lead-scale. **(3) Maintenance risk:** adopting a maintenance-frozen, effectively single-maintainer backbone (§3) as core infrastructure is a strategic liability. On the architecture open question — *adopt SpiderFoot vs. build a lightweight custom orchestrator* — the evidence points clearly to a **lightweight custom orchestrator**: we need to invoke ~6 known modules in a defined order with our own record schema, retries, and audit logging, which is a modest amount of glue code, not a 200-module recursive engine. The two points (not one) reflect that SpiderFoot remains genuinely useful as a *design reference* and as an on-demand deep-dive tool for a suspicious lead, just not as the pipeline backbone.

## 8. Recommendation

**Reference only** (with a narrow optional exception).

**Reasoning + concrete next step:** Do **not** adopt SpiderFoot as the orchestration backbone. Its publisher/subscriber module architecture is worth studying, but its event-graph output, maximal-fan-out philosophy, ToS/GDPR exposure at scale (§6), and maintenance-frozen single-maintainer status (§3) make it wrong for a consent-based, cost-controlled, record-oriented lead pipeline. Build a **lightweight custom orchestrator** for Stage 2 that implements the [architecture module contract](https://github.com/Shinydev09/osint-lead-platform/blob/master/docs/architecture.md) directly (record-in/record-out, per-field `unknown` fallback, per-call legal-basis audit logging) and calls the small approved tool set from the [README pipeline table](https://github.com/Shinydev09/osint-lead-platform/blob/master/README.md).

Concrete next steps:
1. **Reference deliverable:** in the Stage-2 orchestrator design doc, adopt SpiderFoot's `watchedEvents`/`producedEvents`/`handleEvent` plugin interface ([`sfp_dnsresolve.py`](https://github.com/smicallef/spiderfoot/blob/master/modules/sfp_dnsresolve.py)) as the pattern for our `modules/<name>/` interface, and mirror its correlation-rules idea (37 YAML rules) as our downstream risk-flag layer.
2. **Optional narrow exception:** keep an operator-triggered SpiderFoot instance (strict passive module set only, `-u passive`/`-m` allow-list) as a manual deep-dive tool for *individual* suspicious/high-value leads — never in the automated per-lead path — with the legal basis documented per [`docs/compliance.md`](https://github.com/Shinydev09/osint-lead-platform/blob/master/docs/compliance.md). If we don't want to own that operationally, close it out as pure reference.

## Sources

- Repo metadata (license SPDX, stars, open-issues count, `pushed_at`, default branch): [GitHub API — smicallef/spiderfoot](https://api.github.com/repos/smicallef/spiderfoot)
- License text (MIT, "without restriction", copyright holder): [LICENSE](https://github.com/smicallef/spiderfoot/blob/master/LICENSE)
- Features, supported target entities, install instructions, 200+ modules table, HX product split: [README](https://github.com/smicallef/spiderfoot/blob/master/README.md)
- OSS vs. commercial HX feature differences: [spiderfoot.net/open-source-vs-hx](https://www.spiderfoot.net/open-source-vs-hx/)
- Last commit date on master / commit history: [master commits](https://github.com/smicallef/spiderfoot/commits/master)
- Branches (Dependabot-only recent branch): [branches](https://github.com/smicallef/spiderfoot/branches)
- Latest release v4.0 date: [releases](https://github.com/smicallef/spiderfoot/releases)
- Open issues / open PRs: [issues](https://github.com/smicallef/spiderfoot/issues) · [pulls](https://github.com/smicallef/spiderfoot/pulls)
- Contributors: [contributors graph](https://github.com/smicallef/spiderfoot/graphs/contributors)
- Module plugin interface (`watchedEvents`/`producedEvents`/`handleEvent`): [modules/sfp_dnsresolve.py](https://github.com/smicallef/spiderfoot/blob/master/modules/sfp_dnsresolve.py) · [modules/](https://github.com/smicallef/spiderfoot/tree/master/modules)
- Dependency pinning / lxml: [requirements.txt](https://github.com/smicallef/spiderfoot/blob/master/requirements.txt)
- CLI flags (scoping, output formats): [sf.py](https://github.com/smicallef/spiderfoot/blob/master/sf.py)
- Real scan input/output (CSV + JSON) and module count (230 installed): captured locally from SpiderFoot 4.0.0 during this evaluation (`python3 ./sf.py -s example.com -m sfp_dnsresolve,sfp_dnsraw`)
- Platform pipeline stages / candidate tools: [README.md](https://github.com/Shinydev09/osint-lead-platform/blob/master/README.md) · [docs/research/osint-tooling-research.md](https://github.com/Shinydev09/osint-lead-platform/blob/master/docs/research/osint-tooling-research.md)
- Module contract & orchestration open question: [docs/architecture.md](https://github.com/Shinydev09/osint-lead-platform/blob/master/docs/architecture.md)
- GDPR / ToS scope constraints: [docs/compliance.md](https://github.com/Shinydev09/osint-lead-platform/blob/master/docs/compliance.md)
