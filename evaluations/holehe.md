# Evaluation: holehe

- **Repo:** [megadose/holehe](https://github.com/megadose/holehe)
- **Target module:** `email-validate`
- **Evaluator:** Claude (AI research contributor)
- **Date:** 2026-07-13

## 1. Summary

holehe checks whether an email address is registered on 120+ third-party sites (Twitter, Instagram, Amazon, GitHub, etc.) by abusing each site's registration, login, or forgotten-password endpoints, without sending any alert to the target email ([README summary](https://github.com/megadose/holehe#summary)). For sites that expose it, it also returns partially-obfuscated recovery emails and phone numbers ([Module Output](https://github.com/megadose/holehe#module-output)). It ships both a CLI and an async Python API and is explicitly labelled "Built for educational purposes only" ([README license note](https://github.com/megadose/holehe#-license)).

## 2. License

- **License:** GNU General Public License v3.0 (GPL-3.0). The repo's [`LICENSE.md`](https://raw.githubusercontent.com/megadose/holehe/master/LICENSE.md) contains the full verbatim GPLv3 text, GitHub reports the SPDX id as `GPL-3.0` ([license API](https://api.github.com/repos/megadose/holehe/license)), and [`setup.py`](https://github.com/megadose/holehe/blob/master/setup.py) declares the classifier `License :: OSI Approved :: GNU General Public License v3 (GPLv3)`.
- **Commercial/internal-business use allowed without restriction?** **Conditional.** GPL-3.0 permits commercial and internal use, but it is a strong copyleft license: if we distribute a derivative work (or link holehe's GPL code into a distributed product), the combined work must also be released under GPL-3.0 with source. Running it internally as an unmodified, separate process (SaaS/internal tooling that we do not distribute) does **not** trigger the distribution obligation — unlike AGPL, GPL-3.0 has no network-use clause. See the [GPLv3 terms](https://www.gnu.org/licenses/gpl-3.0.en.html).
- **AGPL / "no commercial use" style clauses?** No AGPL and no explicit non-commercial clause. **Two flags, though:** (a) GPL-3.0 copyleft would infect our codebase if we import holehe's modules directly (as its [Python example](https://github.com/megadose/holehe#-python-example) does) into a distributed product — call it as a subprocess or via a service boundary to avoid this; (b) the README states it is "**Built for educational purposes only**" ([README](https://github.com/megadose/holehe#-license)), which is a usage-intent disclaimer that sits awkwardly against production commercial lead-processing.

## 3. Maintenance health

- **Last commit:** 2024-09-10 (`Merge pull request #194 ... Add new module for Duolingo.com`) — roughly 22 months stale as of this evaluation ([commits API](https://api.github.com/repos/megadose/holehe/commits)).
- **Open issues:** 50 open issues, plus 28 open pull requests (78 combined, which is the number GitHub surfaces as `open_issues_count`) ([issue search](https://github.com/megadose/holehe/issues), [repo API](https://api.github.com/repos/megadose/holehe)).
- **Contributors:** 25 ([contributors API](https://api.github.com/repos/megadose/holehe/contributors)). Effectively a **single-maintainer project** — the repo is authored and gatekept by `megadose`, and community fixes sit in 28 unmerged PRs. **Bus-factor risk: yes.**
- **Release cadence:** No GitHub Releases are published ([releases API](https://api.github.com/repos/megadose/holehe/releases) returns an empty list); versioning happens only on PyPI (currently `1.61`, [`setup.py`](https://github.com/megadose/holehe/blob/master/setup.py)). No cadence to speak of — development has been dormant since Sep 2024. This matters because holehe's accuracy depends on tracking each target site's login/registration flow; stale detection modules silently rot into false negatives/positives.

## 4. Input / output contract

**Input:** a single email address (CLI arg or Python function call). **Output (CLI):** a per-site list where `[+]` = account exists, `[-]` = no account, `[x]` = rate-limited/unknown. **Output (Python API):** a list of dicts, one per module, in the JSON-equivalent shape below.

Real run, installed via `pip3 install holehe` (v1.61) and executed as `holehe test@gmail.com` on Python 3.14 (output truncated for length; 121 modules ran):

```
# input
$ holehe test@gmail.com

# output
********************
   test@gmail.com
********************
[x] about.me
[x] adobe.com
[+] amazon.com
[x] amocrm.com
[+] any.do
[-] archive.org
...
[+] twitter.com
[x] venmo.com
...
[-] zoho.com

[+] Email used, [-] Email not used, [x] Rate limit
121 websites checked in 11.72 seconds
```

Per-module structured output (from the [README "Module Output" section](https://github.com/megadose/holehe#module-output), matching the dict returned by the Python API):

```json
{
  "name": "example",
  "rateLimit": false,
  "exists": true,
  "emailrecovery": "ex****e@gmail.com",
  "phoneNumber": "0*******78",
  "others": null
}
```

## 5. Dependencies & runtime

- **Language / runtime:** Python 3 (README targets 3.7+; verified working under Python 3.14) ([README](https://github.com/megadose/holehe#summary)).
- **Install method:** `pip3 install holehe`; also `git clone` + `python3 setup.py install`, or a provided `docker build`/`docker run` ([Installation](https://github.com/megadose/holehe#%EF%B8%8F-installation)). Runtime deps: `termcolor, bs4, httpx, trio, tqdm, colorama` ([`setup.py`](https://github.com/megadose/holehe/blob/master/setup.py)).
- **Required API keys / accounts:** **None.** No API key or account is needed — it queries target sites directly and anonymously ([README](https://github.com/megadose/holehe#-cli-example)).
- **Expected latency for a single lookup:** For the full 121-site CLI sweep, my run reported **`121 websites checked in 11.72 seconds`** (async via trio/httpx). A single-module Python call (e.g. just `snapchat`) is one HTTP round-trip, sub-second — but note that under load many modules return rate-limit `[x]` rather than a true answer (see §6).

## 6. Rate limits / ToS risk

**This is the highest-risk section and the reason `docs/compliance.md` flags email→registered-account checks as Medium risk** ([compliance table](../docs/compliance.md)).

- **Mechanism is inherently adversarial to the target sites.** holehe determines account existence by probing each site's `register`, `login`, or `password recovery` endpoint — see the `Method` column of the [README Modules table](https://github.com/megadose/holehe#modules). It is using those endpoints for a purpose they were not offered for (account enumeration), which is precisely the behaviour most sites' ToS and anti-automation / anti-account-enumeration controls prohibit.
- **Rate-limiting is pervasive and the documented workaround is evasion.** The README's own Modules table marks many sites with a "Frequent Rate Limit ✔" (google, instagram, imgur, spotify, yahoo, zoho, ebay, patreon, venmo, and ~30 more), and my real run returned `[x]` (rate-limited) for the majority of the 121 sites on a single query from one IP. The README's remediation is literally **"Rate limit? Change your IP."** ([README Module Output note](https://github.com/megadose/holehe#module-output)) — i.e. rotate source IPs to defeat the target site's rate controls. **Deliberately rotating IPs to bypass a site's rate limiting is an explicit ToS-circumvention signal** and must not be done at ad-campaign scale.
- **"Educational purposes only" disclaimer.** The [README license section](https://github.com/megadose/holehe#-license) states it is "Built for educational purposes only," which undercuts any argument that the author sanctioned bulk commercial lead validation.
- **Mitigating factor:** it "[Does not alert the target email](https://github.com/megadose/holehe/issues/12)," so there is no direct notification/harm to the lead — but that does not resolve the ToS exposure with the *queried* platforms or the GDPR angle.
- **GDPR / platform-ToS posture for this repo:** This maps directly onto `docs/compliance.md` Hard Rule 3 — gray-zone account-discovery lookups are acceptable only for **spot-checking a small, flagged subset of suspicious leads** under a documented "legitimate interest / anti-fraud" basis (GDPR Art. 6(1)(f)), **never bulk-run against every lead by default** ([compliance hard rules](../docs/compliance.md)). Any adoption must be rate-limited from *our* side, must not IP-rotate to evade target limits, and must log the legal basis and source-permission reference per run.

## 7. Fit score (1-5)

**Score:** 2

**Justification:** In the [README pipeline table](../README.md) holehe is listed under **Validate → `modules/email-validate`** as a candidate alongside AfterShip email-verifier. For that stage the primary question is "is this email real, deliverable, and low-risk?" — and holehe answers a *different* question ("is this email tied to accounts on consumer platforms?"). As the research doc frames it, holehe is at best a **secondary** signal ("this lead's email is tied to a real, active online identity"), while AfterShip email-verifier is the intended core deliverability validator ([osint-tooling-research §3](../docs/research/osint-tooling-research.md)). It scores low, not because it doesn't work — it installed cleanly and ran in ~12s — but because (a) its account-enumeration mechanism is exactly the Medium-risk, ToS-gray behaviour compliance restricts to a *small flagged subset*, which is incompatible with validating **every** lead from an ad campaign; (b) pervasive rate-limiting makes its per-site answers unreliable at any volume without IP rotation we are explicitly barred from doing; and (c) it is dormant (last commit 2024-09-10, single maintainer, 28 unmerged PRs), so detection modules will silently decay. It is a useful **anti-fraud spot-check tool**, not a pipeline-default email validator.

## 8. Recommendation

**Reference only** (with a narrow, gated exception for anti-fraud spot-checks).

**Reasoning + concrete next step:** Do **not** adopt holehe as the default `email-validate` engine — that role belongs to AfterShip email-verifier (syntax/MX/SMTP, no third-party ToS exposure, MIT). holehe's value is a *secondary, selective* anti-fraud signal, but its account-enumeration method, the "change your IP" evasion guidance, the "educational purposes only" disclaimer, GPL-3.0 copyleft, and dormant single-maintainer status all argue against baking it into the always-on pipeline. **Concrete next step:** shelve holehe behind the `AfterShip/email-verifier` evaluation; if a later anti-fraud need materialises, revisit it as an *opt-in, human-triggered* check on individually flagged suspicious leads only — invoked as an isolated subprocess/service (to avoid GPL-3.0 infecting our code), hard rate-limited from our side, no IP rotation, with the GDPR Art. 6(1)(f) legitimate-interest basis and source-permission reference logged per run, and add a corresponding row/notes to `docs/compliance.md` before any `modules/` code lands. Also compare against `MailAccess` (the no-API-key alternative flagged for a bake-off in the research doc) before committing.

## Sources

- Repo, stars, license SPDX, default branch, open-issue count: [megadose/holehe repo API](https://api.github.com/repos/megadose/holehe) / [repo page](https://github.com/megadose/holehe)
- Summary, install, CLI/Python examples, module output format, "change your IP", "educational purposes only", Modules/Method/rate-limit table: [holehe README](https://github.com/megadose/holehe#readme)
- Full GPLv3 license text: [LICENSE.md](https://raw.githubusercontent.com/megadose/holehe/master/LICENSE.md); [GPL-3.0 terms](https://www.gnu.org/licenses/gpl-3.0.en.html)
- Dependencies, version 1.61, license classifier: [setup.py](https://github.com/megadose/holehe/blob/master/setup.py)
- Last commit date/message: [commits API](https://api.github.com/repos/megadose/holehe/commits)
- Contributor count: [contributors API](https://api.github.com/repos/megadose/holehe/contributors)
- No GitHub releases: [releases API](https://api.github.com/repos/megadose/holehe/releases)
- Does-not-alert-target claim: [holehe issue #12](https://github.com/megadose/holehe/issues/12)
- Input/output example: real local run of `holehe test@gmail.com` (holehe v1.61, installed via `pip3 install holehe`, Python 3.14) on 2026-07-13
- Pipeline fit, compliance basis: [this repo's README pipeline table](../README.md), [docs/compliance.md](../docs/compliance.md), [docs/research/osint-tooling-research.md §3 & §11](../docs/research/osint-tooling-research.md)
