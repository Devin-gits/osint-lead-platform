# Evaluation: Maigret

- **Repo:** [github.com/soxoj/maigret](https://github.com/soxoj/maigret)
- **Target module:** `social-footprint` (Validate — "is this a real, active person?")
- **Evaluator:** Claude (AI research contributor)
- **Date:** 2026-07-13

## 1. Summary

Maigret collects a "dossier" on a person **by username only**, checking 3,000+ sites for accounts and scraping available profile data (name, location, bio, linked accounts) from each hit ([README](https://github.com/soxoj/maigret/blob/main/README.md)). It requires no API keys, performs recursive search on newly-discovered usernames/IDs, and exports results as console text, JSON, CSV, HTML, PDF, XMind, or a graph ([README — Usage](https://github.com/soxoj/maigret/blob/main/README.md)). It is the more feature-rich sibling of Sherlock, adding data extraction and report generation on top of pure username enumeration.

## 2. License

- **License:** MIT License ([LICENSE file](https://github.com/soxoj/maigret/blob/main/LICENSE), verified via GitHub API — `spdx_id: MIT`, `Copyright (c) 2020-2026 Soxoj`).
- **Commercial/internal-business use allowed without restriction?** **Yes.** MIT grants use, copy, modify, merge, publish, distribute, sublicense, and sell without restriction beyond retaining the copyright notice ([LICENSE](https://github.com/soxoj/maigret/blob/main/LICENSE)). The README states plainly: *"The open-source Maigret is MIT-licensed and free for commercial use without restriction"* ([README — Commercial Use](https://github.com/soxoj/maigret/blob/main/README.md)).
- **AGPL / "no commercial use" style clauses?** **None.** No AGPL, GPL, or non-commercial clause is present — this is a permissive MIT license with no copyleft obligation. **Note the paid tier is a *separate* offering, not a license restriction:** the maintainer sells a private daily-updated site database (5,000+ sites) and a hosted username-check API, but neither gates the open-source MIT code ([README — Commercial Use](https://github.com/soxoj/maigret/blob/main/README.md)). This is the cleanest license posture of the three `social-footprint` candidates (Social-Analyzer is AGPL-3.0 per the [research doc](../docs/research/osint-tooling-research.md)).

## 3. Maintenance health

- **Last commit:** 2026-07-11 (commit `03a9e2e`, via [GitHub commits API](https://github.com/soxoj/maigret/commits/main)) — active within 2 days of evaluation.
- **Open issues:** **27** open issues (excluding PRs); 7 open PRs. GitHub's raw `open_issues_count` of 34 conflates the two (queried via GitHub search API `repo:soxoj/maigret+type:issue+state:open`). See [issues](https://github.com/soxoj/maigret/issues).
- **Contributors:** **67** contributors ([contributors API](https://github.com/soxoj/maigret/graphs/contributors)). Single-maintainer risk? **Partial** — the project is clearly driven by one lead maintainer (soxoj, sole copyright holder and commercial-offering owner), but 67 contributors and 2,686 forks provide a meaningful bus-factor cushion. Bus factor is a moderate, not acute, concern.
- **Release cadence:** Frequent tagged releases — v0.6.2 (2026-07-01), v0.6.1 (2026-05-16), v0.6.0 (2026-04-10) ([releases](https://github.com/soxoj/maigret/releases)). Roughly monthly. 35,298 stars ([repo](https://github.com/soxoj/maigret)).

## 4. Input / output contract

**Input:** one or more usernames (strings) as CLI args or via the embeddable Python async API ([README — Python library](https://github.com/soxoj/maigret/blob/main/README.md)).
**Output:** per-site account status + extracted profile fields, in the chosen format.

Real run performed during this evaluation (`maigret 0.6.2`, Python 3.14.3, installed via `pip install maigret`):

```
# input
$ maigret soxoj --top-sites 15 --no-progressbar

# output (abridged — real console output)
[*] DB auto-update: downloading database (3187 sites)...
[+] Using sites database: /home/user/.maigret/data.json (3187 sites)
[-] Starting a search on top 23 sites from the Maigret database...
[*] Checking username soxoj on:
[+] GitHub: https://github.com/soxoj
 ├─uid: 31013580
 ├─created_at: 2017-08-14T17:03:07Z
 ├─location: Amsterdam, Netherlands
 ├─follower_count: 2104
 ├─fullname: Soxoj
 ├─twitter_username: sox0j
 ├─bio: CPO @ Social Links
 ├─company: Social Links
 └─_extractor: GitHub API
[+] TikTok: https://www.tiktok.com/@soxoj
[!] Too many errors of type "Access denied" (8.7%). It's recommended to use --cloudflare-bypass or proxy
[-] Extracted IDs: {'sox0j': 'username', 'soxoj': 'username'}
[*] Checking username sox0j on:        # <- recursive search on discovered ID
[+] Twitter: https://twitter.com/sox0j
 ├─fullname: Soxoj
 ├─bio: OSINT enthusiast. ... CPO @ Social Links.
 ├─created_at: 2018-08-19 15:22:14+00:00
 └─location: Amsterdam, Netherlands
```

Structured export from a real `--json simple` run (`maigret soxoj --site GitHub --no-recursion --json simple`):

```json
{
  "GitHub": {
    "http_status": 200,
    "ids_usernames": {"sox0j": "username"},
    "url_main": "https://www.github.com/",
    "url_probe": "https://api.github.com/users/soxoj",
    "status": {
      "site_name": "GitHub",
      "status": "Claimed",
      "url": "https://github.com/soxoj",
      "username": "soxoj",
      "tags": ["coding"],
      "ids": {
        "uid": "31013580",
        "fullname": "Soxoj",
        "bio": "CPO @ Social Links",
        "company": "Social Links",
        "location": "Amsterdam, Netherlands",
        "follower_count": "2104",
        "created_at": "2017-08-14T17:03:07Z",
        "twitter_username": "sox0j",
        "_extractor": "GitHub API"
      }
    }
  }
}
```

Contract, in pipeline terms: **username → map of `{site: {status: Claimed/Available, url, extracted profile fields, discovered linked IDs}}`.** The `ids_usernames` field is what drives recursive pivoting and is the natural key for cross-linking a lead's identities.

## 5. Dependencies & runtime

- **Language / runtime:** Python **3.10+** ([README badge/text](https://github.com/soxoj/maigret/blob/main/README.md)). Verified running on Python 3.14.3 during this evaluation.
- **Install method:** `pip install maigret` (PyPI); also from source (`pip3 install .`), Docker (`soxoj/maigret:latest` CLI, `soxoj/maigret:web` UI), and a standalone Windows `.exe` ([README — Installation](https://github.com/soxoj/maigret/blob/main/README.md)). PDF export is an optional extra (`pip install 'maigret[pdf]'`) needing system graphics libs.
- **Required API keys / accounts:** **None** for core username search ([README](https://github.com/soxoj/maigret/blob/main/README.md)). `OPENAI_API_KEY` is optional, only for the `--ai` summary mode.
- **Expected latency for a single lookup:** No single documented number. Measured in this evaluation: the top-~15/23-site default run with recursion ran for **well over 60s** (recursion fanned out to discovered IDs and I timed it out at 180s); a constrained single-site (`--site GitHub`) run completed in **under ~90s including a first-run 3,187-site DB download**. Latency scales with site count (`-a` = all 3,000+ sites is minutes) and is dominated by network round-trips + per-site timeouts. The DB auto-updates once per 24h ([README — Main features](https://github.com/soxoj/maigret/blob/main/README.md)), adding a one-time download cost.

## 6. Rate limits / ToS risk

**This is the primary risk for this tool.** A single lookup fans out HTTP requests to **thousands of third-party sites** (default run checks the top ~500 by traffic; `-a` scans all 3,000+) — [README — Main features](https://github.com/soxoj/maigret/blob/main/README.md). During the real run above, Maigret itself emitted:

> `[!] Too many errors of type "Access denied" (8.7%). It's recommended to use --cloudflare-bypass or proxy`
> `[!] Too many errors of type "Bot protection" (4.35%). Try to switch to another ip address`

i.e. the tool is already being actively blocked by anti-bot/WAF protections on the sites it probes, and the maintainer's recommended workaround is rotating **residential proxies** (the README is sponsored by three residential-proxy vendors and links a proxy service in the error message itself) — [README — Sponsors / Cloudflare bypass](https://github.com/soxoj/maigret/blob/main/README.md). Routing lead-validation traffic through residential proxies to evade blocks is itself a ToS-circumvention posture that must not be adopted silently.

Maigret does **not** publish a rate-limit policy; ToS exposure is inherited from each of the 3,000+ downstream sites, most of whose ToS prohibit automated access/scraping. The README's own **Disclaimer** states: *"For educational and lawful purposes only. You are responsible for complying with all applicable laws (GDPR, CCPA, etc.) in your jurisdiction. The authors bear no responsibility for misuse."* ([README — Disclaimer](https://github.com/soxoj/maigret/blob/main/README.md)).

This maps directly onto the risks this repo's own [`docs/research/osint-tooling-research.md` §11](../docs/research/osint-tooling-research.md) flags: running platform-querying OSINT "at ad-campaign scale against every lead" is the risky mode, and pulls in **third-party personal data the lead never consented to** (profile fields, linked accounts of the person). Under GDPR (this repo's operator is Portugal-based, per §11) this requires a documented Art. 6 legal basis (legitimate interest / anti-fraud), strict rate-limiting, and must be scoped to spot-checking flagged leads — **not bulk enrichment of every lead**.

## 7. Fit score (1-5)

**Score:** 4

**Justification:** Maigret maps precisely onto the `Validate → modules/social-footprint` row of the [README pipeline table](../README.md) — "is this a real, active person?" It is the research doc's own designated **"Best default choice for the identity-validation module"** ([research doc §5](../docs/research/osint-tooling-research.md)). Against the actual pipeline it earns a 4, not a 5, on a real cost/benefit basis: (a) it takes a *username*, but the pipeline's raw lead is `name, email, phone, company, domain` — there is **no username field**, so Maigret only fits after an upstream step derives a candidate handle (e.g. email local-part, or a handle from `holehe`/`theHarvester`), making it a second-stage confidence signal rather than a primary validator; (b) its structured JSON (`status: Claimed` + extracted `fullname`/`location`/`company`) is genuinely useful as a "this identity is real and active" signal and cross-checks against the lead's declared company/name; (c) the MIT license is the cleanest of the three candidates and it is embeddable as a Python library, so it drops into an async pipeline without shelling out. The one point off is the ToS/GDPR risk in §6 — it is only safe here as a **rate-limited, per-lead, documented-legal-basis spot check**, never bulk, which caps its role in an automated enrichment pipeline.

## 8. Recommendation

**Fork & modify** (adopt the library, not the default behavior).

**Reasoning + concrete next step:** The MIT license, active maintenance, no-API-key operation, embeddable async Python API, and clean structured JSON make Maigret the strongest of the three `social-footprint` candidates and worth adopting — but **not "as-is"**, because its defaults (fan-out to 500–3,000 sites, recursive pivoting, proxy-based block evasion) are exactly the "bulk non-consensual scraping" pattern this repo's [compliance notes §11](../docs/research/osint-tooling-research.md) restrict. Adopt it as an embedded library with a constrained wrapper, not the raw CLI.

Concrete next step for Stage 2 (`modules/social-footprint/`): build a thin wrapper around Maigret's Python library API that (1) accepts a **single derived username** per flagged lead (never bulk), (2) hard-limits site scope via `--tags`/a curated allow-list of ToS-tolerant sites rather than `-a`, (3) disables recursive search by default, (4) **does not** enable residential-proxy/`--cloudflare-bypass` block evasion, and (5) logs the Art. 6 legal basis + permission reference per lookup, with a data-retention/deletion hook, per [compliance requirements](../docs/research/osint-tooling-research.md). Pin the version (currently `0.6.2`) and vendor the site DB to avoid the per-run auto-download at scale.

## Sources

- [Maigret repository](https://github.com/soxoj/maigret) — stars (35,298), forks (2,686), language, default branch, description (GitHub API).
- [Maigret LICENSE file](https://github.com/soxoj/maigret/blob/main/LICENSE) — MIT, Copyright (c) 2020-2026 Soxoj (content verified via GitHub contents API).
- [Maigret README](https://github.com/soxoj/maigret/blob/main/README.md) — features (3,000+ sites, 500 default), install methods, usage/output formats, Python 3.10+, Commercial Use, Sponsors (residential proxies), Disclaimer, Cloudflare bypass.
- [Maigret commits](https://github.com/soxoj/maigret/commits/main) — last commit 2026-07-11 (`03a9e2e`).
- [Maigret issues](https://github.com/soxoj/maigret/issues) — 27 open issues / 7 open PRs (GitHub search API).
- [Maigret contributors](https://github.com/soxoj/maigret/graphs/contributors) — 67 contributors.
- [Maigret releases](https://github.com/soxoj/maigret/releases) — v0.6.2 (2026-07-01), v0.6.1, v0.6.0.
- Real install + run performed during this evaluation: `pip install maigret` → `maigret 0.6.2` on Python 3.14.3; console + `--json simple` outputs captured directly.
- [This repo — README pipeline table](../README.md) and [docs/research/osint-tooling-research.md §5, §11](../docs/research/osint-tooling-research.md) — module mapping and compliance/ToS risk gates.
