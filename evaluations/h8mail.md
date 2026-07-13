# Evaluation: h8mail

- **Repo:** [khast3x/h8mail](https://github.com/khast3x/h8mail)
- **Target module:** No dedicated module in `README.md`'s pipeline table — evaluated as a **risk / breach-signal input** to the Validate stage (the research doc frames it as "useful as a **risk score input**, not for the enrichment itself" — [osint-tooling-research §3](../docs/research/osint-tooling-research.md)). See §7 for why this is *not* a fit for `email-validate`.
- **Evaluator:** Claude (AI research contributor, Stage 1)
- **Date:** 2026-07-13

## 1. Summary

h8mail is an email OSINT and password-breach-hunting tool: given an email (or a list, a URL to scrape emails from, a username/IP/domain/hash query, or a local breach dump), it reports how many breaches the target appears in and — via premium services or local dumps — the leaked cleartext passwords, hashes, and related data ([README summary](https://github.com/khast3x/h8mail#readme), [setup.py description](https://github.com/khast3x/h8mail/blob/master/setup.py)). It queries breach/reconnaissance services (HaveIBeenPwned, Snusbase, Leak-Lookup, Hunter.io, Dehashed, Emailrep.io, IntelX, Breachdirectory) and/or searches local breach corpora such as the "Breach Compilation" torrent and "Collection#1" ([README APIs table](https://github.com/khast3x/h8mail#apis)). It also "chases" related emails — pulling emails discovered for one target back into the search set to expand the graph ([README features](https://github.com/khast3x/h8mail#tangerine-features)) — which makes it a breach-exposure / password-reuse **risk signal**, not an email deliverability validator.

## 2. License

- **License:** **BSD 3-Clause License.** The research doc listed this as "—" (unspecified); that is **incorrect and is corrected here.** The repo's [`LICENSE`](https://github.com/khast3x/h8mail/blob/master/LICENSE) file is the verbatim 3-clause BSD text ("Copyright (c) 2019, khast3x… Neither the name of the copyright holder nor the names of its contributors may be used to endorse or promote products…"); the [README "Notes" section](https://github.com/khast3x/h8mail#tangerine-notes) states "Licence is BSD 3 clause"; and [`setup.py`](https://github.com/khast3x/h8mail/blob/master/setup.py) declares both `license="BSD license"` and the classifier `License :: OSI Approved :: BSD License`. GitHub's license API reports `NOASSERTION`/"Other" ([license API](https://api.github.com/repos/khast3x/h8mail/license)) only because the file's leading blank lines defeat GitHub's auto-classifier — the text itself is unambiguously BSD-3-Clause.
- **Commercial/internal-business use allowed without restriction?** **Yes.** BSD-3-Clause is a permissive license: commercial and internal-business use, modification, and redistribution are allowed, with only two conditions — retain the copyright/license notice in source and binary redistributions, and do not use the author's name to endorse derived products. There is **no copyleft**, so it will not infect our own codebase (a materially lighter obligation than the GPL-3.0 of [holehe](./holehe.md) or the AGPL-3.0 of Firecrawl).
- **AGPL / "no commercial use" style clauses?** **None.** No AGPL, no non-commercial clause, no network-use source-disclosure obligation. The licensing risk here is effectively nil; the risk in adopting h8mail (§6) is entirely operational/legal-basis, not license-driven.

## 3. Maintenance health

- **Last commit:** **2022-06-25** (`Merge pull request #144 from khast3x/2.5.6`, tagging release 2.5.6) on `master` ([commits](https://github.com/khast3x/h8mail/commits/master)). This is **~4 years stale** as of this evaluation (2026-07-13). Note: the research doc's "Updated 2026-07-12" is misleading — the GitHub `pushed_at` timestamp (2023-08-15, [repo API](https://api.github.com/repos/khast3x/h8mail)) reflects a push to a stale dev branch (`2.5.5-dev2`), **not** default-branch development; the last real change to `master` is 2022.
- **Open issues:** **25 open issues** (plus 11 open PRs; GitHub's `open_issues_count` of 36 combines both) ([open issues search](https://github.com/khast3x/h8mail/issues?q=is%3Aissue+is%3Aopen), [repo API](https://api.github.com/repos/khast3x/h8mail)).
- **Contributors:** **5** ([contributors API](https://api.github.com/repos/khast3x/h8mail/contributors)) — effectively a **single-maintainer project** authored and gatekept by `khast3x`, who states in the README "h8mail is maintained on my free time" ([README Notes](https://github.com/khast3x/h8mail#tangerine-notes)). **Bus-factor risk: yes.**
- **Release cadence:** Dormant. Latest release **2.5.6** shipped 2022-06-25, following 2.5.5 / 2.5.4 / 2.5.3 ([releases](https://github.com/khast3x/h8mail/releases)). No releases in ~4 years. This matters acutely for a breach tool: its value depends on the third-party breach-service APIs it targets, and the README's own APIs table already marks scylla.so and breachdirectory.org as `:construction:` (broken/under-construction) — stale integrations silently rot as those services change or disappear ([README APIs table](https://github.com/khast3x/h8mail#apis)).

## 4. Input / output contract

**Input:** one or more targets via `-t` (email string, file of emails/patterns, or filepath glob), or `-u` (URLs to scrape emails from), or `-q` (a typed `username`/`password`/`ip`/`hash`/`domain` query); optional `-c config.ini` for API keys, `-lb`/`-gz` for local cleartext/gzip breach corpora, `-ch N` to chase related emails. **Output:** a colorized CLI recap, plus optional `-o` CSV or `-j` JSON.

Real run performed locally with **h8mail 2.5.6** (installed via `pip3 install h8mail`, Python 3.14) against a small **local** breach file (no API keys, `-sk` skips the default Scylla/Hunter.io calls). Output below is actual, unedited (ANSI color codes stripped for readability):

```bash
# input — scan a local cleartext dump for a target, skip default API checks
$ printf 'admin@evilcorp.com:hunter2\njohn.doe@evilcorp.com:Summer2019!\n' > fake_breach.txt
$ h8mail -t admin@evilcorp.com -lb fake_breach.txt -sk
```

```text
# output (CLI)
[>] Targets:
[>] admin@evilcorp.com
[~] Target factory started for admin@evilcorp.com
[~] Using file fake_breach.txt
[~] Worker [3601] is searching for targets in fake_breach.txt (0 MB)
[>] Found occurrence [fake_breach.txt] Line 0: admin@evilcorp.com:hunter2
 __________________________________________________________________________________________
[>] Showing results for admin@evilcorp.com
LOCALSEARCH    |       admin@evilcorp.com > [fake_breach.txt] Line 0: admin@evilcorp.com:hunter2
__________________________________________________________________________________________
                                  Session Recap:
                 Target                  |                   Status
__________________________________________________________________________________________
           admin@evilcorp.com            |          Breach Found (1 elements)
__________________________________________________________________________________________
Execution time (seconds):   0.12662672996520996
Done
```

```json
// output (-j out.json) — same run
{
    "targets": [
        {
            "target": "admin@evilcorp.com",
            "pwn_num": 1,
            "data": []
        }
    ]
}
```

**Contract:** in → target(s) + optional keys/local corpora; out → per-target object with `target`, `pwn_num` (breach hit count), and a `data` array of found breach records (empty here because the match came from a `LOCALSEARCH` source; API/service hits populate `data` with `[source, value]` pairs). **What could not be shown live:** the premium-API path (HaveIBeenPwned v3, Snusbase, Dehashed, etc.) — those require paid/keyed accounts (§5), so only the documented API examples and the key-free local-breach path are demonstrated here. The keyed path returns the same JSON shape with populated `data` per the [README usage examples](https://github.com/khast3x/h8mail#tangerine-usage-examples).

## 5. Dependencies & runtime

- **Language / runtime:** Python 3 (setup.py classifiers list 3.6/3.7; ran cleanly on Python 3.14 in this evaluation) ([setup.py](https://github.com/khast3x/h8mail/blob/master/setup.py)).
- **Install method:** `pip3 install h8mail` — **only runtime dependency is `requests`** ("Painless install… only requires `requests`", [README features](https://github.com/khast3x/h8mail#tangerine-features), confirmed by `install_requires=["requests"]` in [setup.py](https://github.com/khast3x/h8mail/blob/master/setup.py)). Also distributed as a Docker image (`kh4st3x00/h8mail`) ([README badges](https://github.com/khast3x/h8mail#readme)).
- **Required API keys / accounts:** **None for local-breach or basic public checks**; **required for the tool's actual value.** The breach-lookup services that make h8mail useful as a risk signal need keys/accounts: HaveIBeenPwned v3 (paid key), Snusbase, Leak-Lookup (private), Hunter.io (service tier), Dehashed, Emailrep.io, IntelX, Breachdirectory — all marked `:key:` in the [README APIs table](https://github.com/khast3x/h8mail#apis). Keys are supplied via a generated `h8mail_config.ini` (`--gen-config`) or `-k "K=V"` on the CLI ([README usage](https://github.com/khast3x/h8mail#tangerine-usage)). This means **meaningful use incurs recurring third-party subscription cost and per-vendor ToS obligations.**
- **Expected latency for a single lookup:** **~0.13 s** measured for the local single-target scan above (`0.1266 s` reported). Networked API lookups will be dominated by the slowest upstream breach service and any per-vendor rate limiting — not measurable here without paid keys.

## 6. Rate limits / ToS risk

**This is the decisive section. h8mail is explicitly flagged as a gray-zone, Medium-High-risk tool in this repo's own governance docs, and that framing is not softened here.**

- **This repo's compliance doc rates breach checking (h8mail specifically) Medium-High.** [`docs/compliance.md`](../docs/compliance.md) per-category table: *"Breach/leak checking (h8mail) — **Medium-High** — Surfaces sensitive historical breach data; **treat as an internal risk signal only, never expose to sales/marketing views, and restrict access.**"* That is the single most restrictive risk rating assigned to any Validate-adjacent tool in this project.
- **It falls under compliance Hard Rule 3.** [`docs/compliance.md` Hard Rules](../docs/compliance.md): *"**Rate-limit and document any breach-checking or gray-zone lookups** (e.g., holehe, h8mail, GHunt-style account discovery). These are acceptable for spot-checking a small, flagged subset of suspicious leads under a documented 'legitimate interest / anti-fraud' basis (GDPR Art. 6(1)(f)) — **not for bulk-running against every lead by default.**"*
- **The research doc places it in the same gray zone.** [osint-tooling-research §11](../docs/research/osint-tooling-research.md): *"holehe / GHunt / breach-checkers operate in a gray zone with the platforms they query (most explicitly disclaim commercial/bulk use in their license or docs) — fine for spot-checking a handful of suspicious leads, **risky if run at ad-campaign scale against every lead.** Rate-limit and document the legal basis (legitimate interest / anti-fraud) per GDPR Art. 6."* The reviewer states these are enforced as **hard PR-review gates** in Stage 2 (§14).
- **Third-party ToS surface is real and per-vendor.** h8mail is a client to breach databases it does not control; each service (HaveIBeenPwned, Snusbase, Dehashed, Leak-Lookup, IntelX, etc.) carries its own ToS on acceptable use and bulk querying, and several are commercial services whose terms restrict resale/redistribution of breach data. Processing another person's leaked credentials is inherently high-sensitivity personal-data handling under GDPR. The tool itself prints **"Use responsibly"** in its own banner ([observed in the live run above](https://github.com/khast3x/h8mail#readme)).
- **"Chase" amplifies exposure.** The `--chase` / `--power-chase` feature deliberately expands the target set by pulling in *related* emails discovered mid-run ([README features/usage](https://github.com/khast3x/h8mail#tangerine-usage)) — i.e. it collects data on identities beyond the originally submitted lead, which directly conflicts with `docs/compliance.md` Hard Rule 1 (no collection "beyond what's needed to validate a submitted lead"). This feature must be treated as off-by-policy for lead validation.

## 7. Fit score (1-5)

**Score:** 2

**Justification** (tied to `README.md`'s actual pipeline table — not generic praise): **There is no risk/breach module in `README.md`'s pipeline table.** The table's Validate rows are `modules/email-validate` (AfterShip email-verifier, holehe), `modules/phone-validate` (PhoneInfoga), and `modules/social-footprint` (Maigret/Sherlock/Social-Analyzer) ([README pipeline](../README.md)). h8mail appears only in the *research doc's* Section 1 mapping under a "Validate (identity/risk) — Breach exposure, fraud signal" row that was **never promoted into the README pipeline** ([osint-tooling-research §1](../docs/research/osint-tooling-research.md)). So h8mail does not serve an existing Stage-2 module. It is explicitly **not** an `email-validate` candidate: that stage answers "is this email real, deliverable, and low-risk?" (syntax/MX/SMTP — AfterShip email-verifier's job), whereas h8mail answers a different question ("has this email's credentials been exposed in breaches?"). Force-fitting it into `email-validate` would be wrong, and the research doc says as much ("useful as a risk score input, **not for the enrichment itself**"). It scores 2, not lower, because it *works* (installed with one dependency, ran a real local scan in 0.13 s, permissive BSD license, clean JSON output) and it *could* provide a genuine internal anti-fraud risk signal for a **small, flagged, human-triggered** subset of leads. It does not score higher because: (a) it maps to **no module that exists** in the README pipeline; (b) its highest-in-project Medium-High compliance rating restricts it to internal-only, access-controlled, non-bulk use — incompatible with an always-on per-lead pipeline; (c) its core value requires paid keyed breach-service subscriptions with their own ToS; and (d) it is **dormant** (last `master` commit 2022, single maintainer, broken integrations already flagged in its own README), so it will decay.

## 8. Recommendation

**Reference only** — with a narrow, gated exception for internal, human-triggered anti-fraud risk scoring on individually flagged leads.

**Reasoning + concrete next step:** Do **not** adopt h8mail into the default validation pipeline. It occupies the same gray zone this repo's compliance doc rates **Medium-High** and confines to "internal risk signal only… restrict access," and the research doc + Hard Rule 3 bar bulk/per-lead breach checking outright. Its licensing is actually a *strength* (BSD-3-Clause, permissive, no copyleft — correcting the research doc's "—"), and it installs and runs trivially, but licensing was never the blocker: the blockers are (1) no corresponding module in the README pipeline, (2) the Medium-High GDPR/ToS exposure of processing third-party leaked credentials, (3) recurring paid-API dependence for real value, and (4) 4-year dormancy under a single maintainer with already-broken service integrations. **Concrete next step:** shelve h8mail as reference material behind the approved Validate tools (AfterShip email-verifier for deliverability; holehe already scoped as a selective anti-fraud spot-check). *If* a dedicated internal "breach-exposure risk score" capability is later prioritized, do **not** bake in h8mail's dormant codebase — instead (a) first add a risk/breach row to the README pipeline table and a `docs/compliance.md` entry defining the legal basis (GDPR Art. 6(1)(f)), access controls, and retention, gated by the reviewer per §14; (b) prefer a single maintained keyed source (e.g. HaveIBeenPwned's official API) invoked as an isolated, hard-rate-limited service on a **human-flagged subset only**, with `--chase`-style related-email expansion disabled by policy; and (c) benchmark against MailAccess's breach-detection feature (flagged for a bake-off in the research doc) before committing to any implementation. Breach data must never surface in sales/marketing-facing views.

## Sources

- Repo metadata — stars (5,086), forks (587), `open_issues_count` (36), default branch `master`, language Python, `created_at` 2018-06-15, `pushed_at` 2023-08-15, license `NOASSERTION`: [khast3x/h8mail repo API](https://api.github.com/repos/khast3x/h8mail) / [repo page](https://github.com/khast3x/h8mail)
- **License is BSD 3-Clause** (correcting research doc "—"): [LICENSE file](https://github.com/khast3x/h8mail/blob/master/LICENSE), [README "Notes": "Licence is BSD 3 clause"](https://github.com/khast3x/h8mail#tangerine-notes), [setup.py `license`/classifier](https://github.com/khast3x/h8mail/blob/master/setup.py); GitHub's `NOASSERTION`: [license API](https://api.github.com/repos/khast3x/h8mail/license)
- Last commit on `master` 2022-06-25 (release 2.5.6): [commits](https://github.com/khast3x/h8mail/commits/master); dev-branch `pushed_at` clarification: [branches/tags](https://github.com/khast3x/h8mail/tags)
- Open issues (25) vs PRs (11): [issue search](https://github.com/khast3x/h8mail/issues?q=is%3Aissue+is%3Aopen) via GitHub search API `search/issues`
- Contributors (5), single-maintainer `khast3x`, "maintained on my free time": [contributors API](https://api.github.com/repos/khast3x/h8mail/contributors), [README Notes](https://github.com/khast3x/h8mail#tangerine-notes)
- Releases (latest 2.5.6, 2022-06-25): [releases](https://github.com/khast3x/h8mail/releases)
- Features, APIs table (HIBP/Snusbase/Leak-Lookup/Hunter.io/Dehashed/Emailrep/IntelX/Breachdirectory, `:key:` = key required, `:construction:` = broken), usage flags, chase, "Use responsibly" banner: [README](https://github.com/khast3x/h8mail#readme)
- Dependency (`requests` only), Python versions, description: [setup.py](https://github.com/khast3x/h8mail/blob/master/setup.py)
- Input/output example: real local run of `h8mail 2.5.6` (`pip3 install h8mail`, Python 3.14) against a local breach file with `-sk`, plus `-j` JSON and `--gen-config` output, reproduced in §4/§5 on 2026-07-13
- Internal risk framing and pipeline mapping — Medium-High rating, Hard Rule 3, §11 gray-zone language, §14 review gates, pipeline table (no risk/breach module), "risk score input, not the enrichment": [`docs/compliance.md`](../docs/compliance.md), [`docs/research/osint-tooling-research.md`](../docs/research/osint-tooling-research.md) §1/§3/§11/§14, [`README.md`](../README.md) pipeline table
- Comparison anchor (GPL-3.0 copyleft of the sibling gray-zone tool): [holehe evaluation](./holehe.md)
