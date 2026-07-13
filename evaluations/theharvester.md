# Evaluation: theHarvester

- **Repo:** https://github.com/laramies/theHarvester
- **Target module:** `domain-intel` (Ingest stage — turning a raw lead's domain into subdomains, hosts, IPs, and contact emails; also referenced for `company-enrich` / contact discovery)
- **Evaluator:** research contributor (AI agent, Stage 1)
- **Date:** 2026-07-13

## 1. Summary

theHarvester is a mature Python OSINT CLI built for the reconnaissance stage of a red-team/pentest engagement: given a single domain, it enumerates subdomains, hosts, IPs, emails, and names by querying many public and commercial data sources ([README "About"](https://github.com/laramies/theHarvester/blob/master/README.md)). It is passive-first — most of its ~55 modules are third-party search engines, certificate-transparency logs, and threat-intel APIs, with only DNS brute-force and screenshots as active modules. It ships as a CLI (`theHarvester`) plus an optional FastAPI REST server (`restfulHarvest`), and can emit results to XML/JSON via `-f` ([`-h` output](https://github.com/laramies/theHarvester/blob/master/README.md)).

## 2. License

- **License:** **GNU General Public License v2.0 (GPL-2.0)** — declared verbatim in `README/LICENSES`: *"Released under the GPL v 2.0 … under the terms of the GNU General Public License as published by the Free Software Foundation version 2 of the License"* (Copyright 2011 Christian Martorella) ([README/LICENSES](https://github.com/laramies/theHarvester/blob/master/README/LICENSES)). Note: GitHub's license API returns `null` for this repo because the license text lives at the non-standard path `README/LICENSES` rather than a root `LICENSE` file, so the SPDX id must be read from the file itself (done here).
- **Commercial/internal-business use allowed without restriction?** **Conditional.** GPL-2.0 places **no restriction on *use*** — running the program for any purpose, including commercial and internal business use, is unrestricted, and there is **no "non-commercial" clause**. The restriction is on **distribution**: GPL-2.0 is *strong copyleft*, so if we were to create and **distribute** a derivative work (e.g. import theHarvester's modules into our own codebase and ship that combined product), the combined work would have to be released under GPL-2.0. For our platform this matters because our own code is **MIT** ([root `LICENSE`](../LICENSE)) — mixing GPL-2.0 source into it and distributing would create a copyleft obligation over the combined work.
- **AGPL / "no commercial use" style clauses?** **No AGPL and no non-commercial clause** — this is an important distinction. It is **GPL-2.0, not AGPL-3.0**, so there is **no network-use / SaaS source-disclosure trigger**: running theHarvester behind an internal API does *not* obligate us to release source (unlike AGPL, which several other candidates in the research doc use). The only copyleft exposure is classic GPL distribution, which we avoid entirely if we invoke theHarvester as a **separate CLI subprocess** (mere aggregation) rather than importing it as a library — see the Recommendation. Flagged explicitly per the review checklist ([README/LICENSES](https://github.com/laramies/theHarvester/blob/master/README/LICENSES)).

## 3. Maintenance health

- **Last commit:** 2026-07-06 (`master`, commit "harden github actions more"; repo `pushedAt` 2026-07-06) ([commits](https://github.com/laramies/theHarvester/commits/master)).
- **Open issues:** 4 open issues (the repo's `open_issues_count` of 8 includes 4 open pull requests) — a strikingly low bug backlog for a 15-year-old, widely-used tool ([issues](https://github.com/laramies/theHarvester/issues), verified via `gh api search/issues`).
- **Contributors:** 100+ (the GitHub contributors API returns a full page of 100 without exhausting the list) — **not single-maintainer**; bus-factor risk is low. Three named active maintainers carry the project: Christian Martorella (@laramies, original author), Matthew Brown (@NotoriousRebel1), and Jay Townsend (@jay_townsend1) ([README "Main contributors"](https://github.com/laramies/theHarvester/blob/master/README.md), [contributors graph](https://github.com/laramies/theHarvester/graphs/contributors)). 16,771 stars / 2,525 forks indicate a large, established user base ([repo](https://github.com/laramies/theHarvester)).
- **Release cadence:** Active and steady. Latest release **v4.11.1** published 2026-06-03; project created 2011-01-01 (~15 years old). It is packaged in Kali and many distros ([releases](https://github.com/laramies/theHarvester/releases), [repology badge in README](https://github.com/laramies/theHarvester/blob/master/README.md)).

## 4. Input / output contract

Real run performed locally with theHarvester **v4.11.1** installed from source (`git clone` + `pip install .`), querying the keyless `hackertarget` source. Output below is the actual, unedited program output (truncated to the first rows for length; 50 hosts were returned).

```
# input
$ theHarvester -d owasp.org -b hackertarget

# output (banner omitted)
[*] Target: owasp.org

[*] Searching Hackertarget.

[*] No IPs found.
[*] No emails found.
[*] No people found.

[*] Hosts found: 50
---------------------
20thanniversary.owasp.org:172.66.157.115
aghast.owasp.org:185.199.110.153
aivss.owasp.org:104.20.44.163
asvs.owasp.org:172.66.157.115
cheatsheetseries.owasp.org:172.66.157.115
cloud.owasp.org:157.245.12.71
docs.owasp.org:104.16.254.120
lists.owasp.org:174.138.70.208
mail.owasp.org:104.20.44.163
members.owasp.org:104.20.44.163
...   (50 hosts total, each as  <subdomain>:<IP>)

real    0m6.214s
```

**Contract:** in → a domain (`-d`), a source or set of sources (`-b`, e.g. `hackertarget`, `crtsh`, `bing`, `all`), optional result limit (`-l`, default 500), optional DNS resolution (`-n`), and optional output file (`-f FILENAME` → writes both XML and JSON). out → to stdout, four sections — **IPs**, **emails**, **people (names)**, and **hosts** (`subdomain:ip` pairs); with `-f`, the same data is serialized to `<FILENAME>.json` / `.xml` for machine consumption ([`-h` usage](https://github.com/laramies/theHarvester/blob/master/README.md)). For our pipeline this maps a lead's `domain` field onto (a) a subdomain/host inventory (domain-intel) and (b) discovered `email` values (contact discovery). Note the contact/email yield depends heavily on which sources are enabled — the keyless `hackertarget` run above returned hosts+IPs but no emails; email-rich results generally require API-keyed sources such as Hunter or Tomba (Section 5/6).

## 5. Dependencies & runtime

- **Language / runtime:** Python **3.12 or higher** per the README install section (ran successfully here on Python 3.14). Pulls a substantial dependency set (aiohttp/aiodns async stack, `censys`, `shodan`, `playwright`, `fastapi`, `uvicorn`, `slowapi`, `dnspython`, `tldextract`, etc.) ([README install](https://github.com/laramies/theHarvester/blob/master/README.md); dependency list observed at install time).
- **Install method:** Official path is `git clone` + [`uv`](https://astral.sh/uv) (`uv sync` → `uv run theHarvester`); also installable via `pip install .` from the clone (used here) and via an official **Docker** image / `docker-compose.yml`. It is **not** reliably installable from PyPI as `pip install theHarvester` — that name currently resolves to a stub placeholder package (`theHarvester 0.0.1`, ~97 KB, not the real tool), so source/uv/Docker install is required ([README install](https://github.com/laramies/theHarvester/blob/master/README.md), verified: PyPI `theHarvester` installs v0.0.1 stub whereas the source tree is v4.11.1).
- **Required API keys / accounts:** **None to run** — several modules are fully keyless (`hackertarget`, `crtsh`, `rapiddns`, `duckduckgo`, `otx`, `threatminer`, `thc`, `certspotter`, etc.). **But most high-value sources require paid API keys**: the README lists ~40 key-gated modules with pricing, e.g. Hunter (50 free credits/mo, $34/yr for 12k), Shodan ($69/mo+), SecurityTrails (50 free/mo, $500 for 20k), Censys ($100/500 credits), haveibeenpwned ($4.50+/mo), Tomba, VirusTotal, etc. Keys go in `~/.theHarvester/api-keys.yaml` (auto-created on first run, observed here) ([README "Modules that require an API key"](https://github.com/laramies/theHarvester/blob/master/README.md), [API-keys wiki](https://github.com/laramies/theHarvester/wiki/Installation#api-keys)).
- **Expected latency for a single lookup:** **~6.2 s** measured wall-clock for one domain against a single keyless source (`hackertarget`, 50 hosts) in this run. Multi-source runs (`-b all`) fan out across dozens of network endpoints and take substantially longer and are bounded by the slowest/most-rate-limited source.

## 6. Rate limits / ToS risk

theHarvester's entire model is to query **third-party sites and APIs**, so ToS/rate-limit risk is inherent and non-trivial at scale:

- **Free/keyless sources scraped or hit anonymously** (search engines, `crt.sh`, `hackertarget`, `rapiddns`, `dnsdumpster`) impose their own informal rate limits and, for search engines especially, automated querying can violate their ToS if run at volume. In this evaluation `crt.sh` returned empty responses on repeated back-to-back calls (consistent with its known throttling/instability), while `hackertarget` (which publishes a free-tier query cap) returned results — illustrating that keyless sources are best-effort and rate-limited in practice.
- **API-keyed sources carry explicit, documented quotas and paid tiers** — the README's "Modules that require an API key" section enumerates per-source limits and pricing (e.g. haveibeenpwned "10 email searches/min $4.50", SecurityTrails "50 free queries/month", Shodan paid-only, fofa "10,000/month", etc.) ([README API-key module list](https://github.com/laramies/theHarvester/blob/master/README.md)). Staying within these quotas — and honoring each vendor's ToS — is the integrator's responsibility; there is no built-in global rate governor beyond the optional `slowapi` limiter on the REST server.
- **Compliance fit for our pipeline:** theHarvester is documented as a tool "designed to be used during the reconnaissance stage of a red team assessment or penetration test" ([README "About"](https://github.com/laramies/theHarvester/blob/master/README.md)) — i.e. offensive recon. Our platform only processes leads for which "explicit permission to process that data has been obtained" ([README](../README.md)), and `docs/compliance.md` requires a documented legal basis before OSINT collection. Using theHarvester on a lead's **own** domain (which the lead consented to) is defensible; pointing it at arbitrary third-party domains at scale would raise both vendor-ToS and GDPR legal-basis concerns. Some sources it can query (breach databases like haveibeenpwned/dehashed/leaklookup) are exactly the "bulk breach-checking" category the README flags as out-of-scope/legal-basis-gated — those modules must be **disabled by policy** in any integration.

## 7. Fit score (1-5)

**Score:** 4

**Justification** (connected to `README.md`'s pipeline table): The pipeline table lists theHarvester as a candidate for exactly this stage — `modules/domain-intel` under **Ingest**, alongside web-check — whose job is to take the raw lead's `domain` and enrich the `Raw lead (name, email, phone, company, domain)` record. theHarvester hits that target directly and was proven to do so in the live run above: `-d owasp.org` produced a 50-host subdomain+IP inventory in ~6 s, and its `emails`/`people` output columns feed the `company-enrich`/contact-discovery use it is also referenced for. Structured `-f` JSON output means it can be wrapped behind a module interface without scraping stdout. It is **not a 5** because: (a) it is a **red-team recon CLI, not a lead-enrichment service** — its defaults, source mix (breach DBs, offensive recon), and offensive framing require careful policy gating against our consent/GDPR model; (b) the **most valuable sources are paywalled** (Hunter, SecurityTrails, Shodan…), so keyless-only operation yields mostly host/subdomain data and few contact emails, limiting its standalone enrichment value; and (c) **GPL-2.0 copyleft** constrains how we may integrate it (subprocess/aggregation only, not library import into our MIT code — see below).

## 8. Recommendation

**Fork & modify — no. Adopt as-is via subprocess wrapping — yes, with policy gating.** Net recommendation: **Adopt as-is (reference/CLI-invocation), do not import as a library.**

**Reasoning + concrete next step:** theHarvester is the right shape for `domain-intel` Ingest — mature (15 yrs, 100+ contributors, low bug backlog, active releases), self-hostable, keyless-capable for basic subdomain/host discovery, and empirically produces the exact domain→hosts/IPs/emails contract our Ingest stage needs, with structured `-f` JSON output. The two hard constraints are **license** and **compliance**, and both are manageable if we integrate it the right way: because it is **GPL-2.0 (not AGPL)**, running it as a **separate CLI subprocess** (or the bundled `restfulHarvest` REST server) is *mere aggregation* and imposes **no copyleft obligation on our MIT-licensed code** — whereas importing `theHarvester.*` Python modules into our own package and distributing that would trigger GPL-2.0 over the combined work and is therefore off the table. **Concrete Stage-2 next step:** open a `modules/domain-intel/` implementation PR that (1) invokes theHarvester as an external process via its `restfulHarvest` API or `theHarvester -f out.json`, parsing the JSON — never importing its modules — to preserve the MIT/GPL boundary; (2) pins v4.11.1 and installs from source/Docker (not the PyPI stub); (3) ships an **allowlist of permitted sources** that excludes breach-database modules (haveibeenpwned, dehashed, leaklookup) per `docs/compliance.md`, and restricts targets to lead-owned/consented domains; and (4) adds per-source rate limiting and GDPR legal-basis logging around each call. Pair it with web-check (the other Ingest candidate) for coverage comparison before locking the module's tool choice.

## Sources

- Repo metadata — stars (16,771), forks (2,525), `open_issues_count` (8), created 2011-01-01, `pushedAt` 2026-07-06, language Python, default branch `master`, `license: null` — GitHub API via `gh` CLI: [github.com/laramies/theHarvester](https://github.com/laramies/theHarvester)
- License = GPL-2.0 (full text, "Released under the GPL v 2.0", Copyright 2011 Christian Martorella): [README/LICENSES](https://github.com/laramies/theHarvester/blob/master/README/LICENSES)
- README (About/red-team framing, install via uv/git/Docker, Python 3.12+, passive & active module lists, "Modules that require an API key" with per-source pricing/limits, REST `THEHARVESTER_API_KEY`, maintainers): [README.md](https://github.com/laramies/theHarvester/blob/master/README.md)
- Last commit date 2026-07-06: [commits/master](https://github.com/laramies/theHarvester/commits/master)
- Open issue count = 4 issues (8 open_issues incl. 4 PRs): verified via `gh api search/issues` for `is:issue is:open` and `is:pr is:open` on [issues](https://github.com/laramies/theHarvester/issues)
- Contributors 100+ and named maintainers: [contributors graph](https://github.com/laramies/theHarvester/graphs/contributors), README "Main contributors"
- Latest release v4.11.1 (2026-06-03): [releases](https://github.com/laramies/theHarvester/releases)
- API-key setup and per-source quotas: [Installation wiki – API keys](https://github.com/laramies/theHarvester/wiki/Installation#api-keys)
- Input/output example + ~6.2 s latency: real local run of theHarvester v4.11.1 (`-d owasp.org -b hackertarget`), command + unedited output reproduced in Section 4; `-f`/`-l`/`-b` flags from installed `theHarvester -h`
- PyPI `theHarvester` name resolves to a v0.0.1 stub (not the real tool): verified via `pip install theHarvester` (installed 0.0.1) vs. source tree v4.11.1
- Internal pipeline mapping, MIT license of our own code, and compliance requirements: [`README.md`](../README.md) pipeline table + License section, [`LICENSE`](../LICENSE), [`docs/compliance.md`](../docs/compliance.md), [`docs/research/osint-tooling-research.md`](../docs/research/osint-tooling-research.md)
