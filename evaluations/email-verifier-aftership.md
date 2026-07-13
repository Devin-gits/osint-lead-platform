# Evaluation: AfterShip/email-verifier

- **Repo:** https://github.com/AfterShip/email-verifier
- **Target module:** `email-validate`
- **Evaluator:** Claude (AI research contributor)
- **Date:** 2026-07-13

## 1. Summary

A Go library that verifies email addresses **without sending any email**, layering syntax validation, DNS/MX lookup, SMTP mailbox probing (catch-all detection on by default), and misc checks for disposable, free-provider, and role-account addresses ([README features](https://github.com/AfterShip/email-verifier/blob/main/README.md#features)). It returns a single structured result plus a `reachable` confidence field and optional domain-typo suggestions ([usage](https://github.com/AfterShip/email-verifier/blob/main/README.md#usage)). It is a library-first tool, but ships a reference self-hosted HTTP API server under [`cmd/apiserver`](https://github.com/AfterShip/email-verifier/tree/main/cmd/apiserver) ([API section](https://github.com/AfterShip/email-verifier/blob/main/README.md#api)).

## 2. License

- **License:** MIT License ([LICENSE](https://github.com/AfterShip/email-verifier/blob/main/LICENSE), `Copyright (c) 2020 AfterShip`).
- **Commercial/internal-business use allowed without restriction?** **Yes.** MIT grants use, copy, modify, merge, publish, distribute, sublicense, and sell "without restriction," conditioned only on retaining the copyright/permission notice ([LICENSE](https://github.com/AfterShip/email-verifier/blob/main/LICENSE)). This is compatible with this repo's own MIT license ([README §License](https://github.com/Shinydev09/osint-lead-platform/blob/master/README.md#license)).
- **AGPL / "no commercial use" style clauses?** **None.** No copyleft, no network-use clause, no non-commercial restriction. This is the least restrictive candidate in the `email-validate` row and does not carry the AGPL/GPL obligations flagged for peers like `holehe` (GPL-3.0) in the [tool survey](https://github.com/Shinydev09/osint-lead-platform/blob/master/docs/research/osint-tooling-research.md#3-email-discovery-verification--validation).

## 3. Maintenance health

- **Last commit:** 2025-12-05 (`a3c5c5b`, "Fix should return the domain suggestion when domain is known (#140)"), per [commits API on `main`](https://github.com/AfterShip/email-verifier/commits/main). Note: GitHub's repo `updatedAt` metadata reads 2026-07-08 (matching the survey), but that reflects a metadata touch, not a code push — the last real code change is 2025-12-05.
- **Open issues:** 26 open issues (excluding PRs), via [GitHub issue search](https://github.com/AfterShip/email-verifier/issues?q=is%3Aissue+is%3Aopen); 8 open PRs.
- **Contributors:** 16 ([contributors](https://github.com/AfterShip/email-verifier/graphs/contributors)). **Single-maintainer risk? No** — it is a company-backed (AfterShip) project with a double-digit contributor base, materially lower bus-factor than the single-author peers in this category.
- **Release cadence:** Tagged releases from v1.0.2 through **v1.4.1** (latest, published 2024-09-12), per [releases](https://github.com/AfterShip/email-verifier/releases). Cadence has slowed — ~14 months since the last tagged release and ~7 months since the last commit — so treat it as **stable/mature but low-velocity** rather than actively evolving.

## 4. Input / output contract

**Input:** a single email string to `verifier.Verify(email)`. **Output:** a `Result` struct (JSON-serializable) with syntax breakdown, MX presence, disposable/free/role flags, a `reachable` confidence enum, and (when enabled) SMTP/gravatar/suggestion fields.

Real run below — library installed via `go get github.com/AfterShip/email-verifier@v1.4.1` and executed locally on 2026-07-13. SMTP check left at its default (disabled), matching real production behavior on hosts where outbound port 25 is blocked (see §6), so `smtp` is `null`:

```
# input
verifier := emailverifier.NewVerifier().EnableDomainSuggest()
ret, _ := verifier.Verify("support@github.com")

# output (json.MarshalIndent of ret)
{
  "email": "support@github.com",
  "reachable": "unknown",
  "syntax": {
    "username": "support",
    "domain": "github.com",
    "valid": true
  },
  "smtp": null,
  "gravatar": null,
  "suggestion": "",
  "disposable": false,
  "role_account": true,
  "free": false,
  "has_mx_records": true
}
```

This matches the documented result shape in the [README basic usage example](https://github.com/AfterShip/email-verifier/blob/main/README.md#basic-usage). Note the correct `role_account: true` (support@ is a role address) and `has_mx_records: true` — exactly the deliverability/quality signals the `email-validate` module needs.

## 5. Dependencies & runtime

- **Language / runtime:** Go (module `github.com/AfterShip/email-verifier`); builds cleanly under Go 1.22.5. Transitive deps are minimal: `golang.org/x/net`, `golang.org/x/text`, `github.com/hbollon/go-edlib` (resolved during the `go get` above).
- **Install method:** `go get -u github.com/AfterShip/email-verifier` ([Install](https://github.com/AfterShip/email-verifier/blob/main/README.md#install)); or run the reference [`cmd/apiserver`](https://github.com/AfterShip/email-verifier/tree/main/cmd/apiserver) as a self-hosted microservice.
- **Required API keys / accounts:** **None.** No third-party API keys or accounts required.
- **Expected latency for a single lookup:** **~8 ms measured** for the non-SMTP path (syntax + MX + misc) on the run above. With SMTP enabled, the [FAQ](https://github.com/AfterShip/email-verifier/blob/main/README.md#faq) documents that lookups **hang until a ~30 s timeout** when the ISP blocks outbound port 25 — so real-world SMTP latency is dominated by network/port conditions, not the library.

## 6. Rate limits / ToS risk

The library imposes no rate limits of its own and, by design, **sends no email** ([README title/features](https://github.com/AfterShip/email-verifier/blob/main/README.md#features)). The one third-party-interaction risk is the **SMTP deliverability probe**: with `EnableSMTPCheck()`, it opens a connection to the target domain's mail server and issues `RCPT`-style checks. The README itself notes that "most ISPs block outgoing SMTP requests through port 25 to prevent email spamming" and that the check "will not perform SMTP checking by default" unless explicitly enabled ([Usage note](https://github.com/AfterShip/email-verifier/blob/main/README.md#email-verification-lookup) and [FAQ](https://github.com/AfterShip/email-verifier/blob/main/README.md#faq)). At scale, repeated SMTP probes against a provider can look like address-harvesting/spam reconnaissance and may trip provider abuse thresholds or greylisting — so SMTP mode should be rate-limited and, per this repo's [`docs/compliance.md`](https://github.com/Shinydev09/osint-lead-platform/blob/master/docs/compliance.md) and the survey's [§11 compliance notes](https://github.com/Shinydev09/osint-lead-platform/blob/master/docs/research/osint-tooling-research.md#11-compliance--risk-notes-read-before-agents-start-building), backed by a documented GDPR Art. 6 legitimate-interest / anti-fraud basis when processing lead emails. The syntax/MX/misc path (used in §4) carries no such ToS exposure. A SOCKS5 proxy option exists for SMTP ([proxy usage](https://github.com/AfterShip/email-verifier/blob/main/README.md#use-a-socks5-proxy-to-verify-email)).

## 7. Fit score (1-5)

**Score:** 5

**Justification** (connected to the pipeline table in [`README.md`](https://github.com/Shinydev09/osint-lead-platform/blob/master/README.md#pipeline)): the `Validate → modules/email-validate` row asks "is it real, deliverable, low-risk?" for a raw lead's email. This library answers exactly that in one call — syntax validity, `has_mx_records` (domain can receive mail), `reachable` confidence, plus `disposable`/`free`/`role_account` quality flags that directly feed a lead-quality/risk score — without sending email (a GDPR- and deliverability-friendly property). It is Go, MIT, dependency-light, needs no API keys, and runs in single-digit milliseconds on the non-SMTP path, making it a clean embedded core (or self-hosted API) rather than a paid external dependency. It slots ahead of `holehe`, the other `email-validate` candidate, whose GPL-3.0 license and platform-registration probing put it in the survey's medium-risk bucket; email-verifier is the deliverability primary, with holehe reserved as a secondary "is this identity active" signal.

## 8. Recommendation

**Adopt as-is** (as the primary `email-validate` engine).

**Reasoning + concrete next step if adopted:** MIT license with zero commercial restrictions, no API keys, minimal dependencies, a verified-working structured contract (§4), and a maintenance profile (company-backed, 16 contributors) strong enough for a core module despite slowed release velocity. The one caveat — SMTP probing latency/ToS risk — is opt-in and controllable. **Concrete next step:** in Stage 2, open an implementation PR against `modules/email-validate/` that wraps `verifier.Verify()` with the SMTP check **disabled by default**, exposes `disposable`/`free`/`role_account`/`has_mx_records`/`reachable` as normalized lead-quality fields per the module contract in [`docs/architecture.md`](https://github.com/Shinydev09/osint-lead-platform/blob/master/docs/architecture.md), and gates any opt-in SMTP mode behind a rate limiter plus the documented GDPR legal-basis check from [`docs/compliance.md`](https://github.com/Shinydev09/osint-lead-platform/blob/master/docs/compliance.md). Pin to release `v1.4.1`.

## Sources

- Repo, license badge, features, install, usage, API, FAQ: [README.md](https://github.com/AfterShip/email-verifier/blob/main/README.md)
- License text (MIT, Copyright (c) 2020 AfterShip): [LICENSE](https://github.com/AfterShip/email-verifier/blob/main/LICENSE)
- Last commit `a3c5c5b` 2025-12-05 on main: [commits](https://github.com/AfterShip/email-verifier/commits/main)
- Open issues / PRs: [issues](https://github.com/AfterShip/email-verifier/issues?q=is%3Aissue+is%3Aopen)
- Contributors (16): [graphs/contributors](https://github.com/AfterShip/email-verifier/graphs/contributors)
- Releases (latest v1.4.1, 2024-09-12): [releases](https://github.com/AfterShip/email-verifier/releases)
- Reference API server: [cmd/apiserver](https://github.com/AfterShip/email-verifier/tree/main/cmd/apiserver)
- Input/output example: real local run of `verifier.Verify("support@github.com")` with `email-verifier@v1.4.1` under Go 1.22.5 on 2026-07-13 (output reproduced verbatim in §4); shape matches the [README basic usage example](https://github.com/AfterShip/email-verifier/blob/main/README.md#basic-usage).
- This repo's pipeline table, license, compliance, and tool survey: [README.md](https://github.com/Shinydev09/osint-lead-platform/blob/master/README.md), [docs/compliance.md](https://github.com/Shinydev09/osint-lead-platform/blob/master/docs/compliance.md), [docs/research/osint-tooling-research.md](https://github.com/Shinydev09/osint-lead-platform/blob/master/docs/research/osint-tooling-research.md)
