# Compliance & Risk Notes

Living document — update whenever a new data source or tool is added. This exists because the platform processes personal data (names, emails, phone numbers, company/domain data) collected from ads and business websites under explicit permission from the business running them. Permission from the business is **not** the same as consent from every individual lead, so each technique needs its own legal-basis check.

## Hard rules

1. **No non-consensual personal surveillance.** Tools that pull a person's private social graph, location history, or cross-platform identity beyond what's needed to validate a submitted lead (e.g., reverse-image "find their Instagram/Facebook" tools, follower/location dumping tools) are **out of scope** for the core pipeline. Reference-only unless a specific case gets written legal sign-off.
2. **Respect third-party ToS.** Do not deploy scrapers against platforms whose ToS explicitly prohibits automated access at the scale we'd need (this currently rules out LinkedIn profile scraping as a production data source — reference implementations may be studied, not deployed).
3. **Rate-limit and document any breach-checking or gray-zone lookups** (e.g., holehe, h8mail, GHunt-style account discovery). These are acceptable for spot-checking a small, flagged subset of suspicious leads under a documented "legitimate interest / anti-fraud" basis (GDPR Art. 6(1)(f)) — not for bulk-running against every lead by default.
4. **Log the legal basis and source permission reference** for every enrichment/validation run — which ad campaign or website the lead came from, and the permission record tied to it.
5. **Data retention.** Define and enforce a retention window before storing enriched/validated lead data long-term; leads that fail validation should have a shorter retention default than leads that convert.

## Per-category risk notes

| Category | Risk level | Notes |
|---|---|---|
| Website/domain intel (web-check, theHarvester, subfinder) | Low | Public DNS/WHOIS/tech-stack data about a business's own domain — low personal-data exposure. |
| Email verification (AfterShip email-verifier) | Low | Syntax/MX/SMTP checks, no data sent to third parties about the person. |
| Email → registered-account checks (holehe, MailAccess) | Medium | Queries third-party platforms per email; most disclaim bulk/commercial use in their docs — rate-limit and use selectively. |
| Breach/leak checking (h8mail) | Medium-High | Surfaces sensitive historical breach data; treat as an internal risk signal only, never expose to sales/marketing views, and restrict access. |
| Phone OSINT (PhoneInfoga) | Low-Medium | Carrier/line-type lookups are fairly standard fraud-prevention practice; scam-score lookups may hit rate limits on third-party sources. |
| Social footprint (Sherlock, Maigret, Social-Analyzer) | Medium | Confirms a lead's online presence is real; avoid over-collecting beyond a simple match/no-match + confidence score. |
| Reverse-image / deep account discovery (EagleEye, GHunt) | High | Excluded from default pipeline. Requires case-by-case legal review before any use. |
| LinkedIn scraping | High | Excluded from production. ToS violation with real legal precedent; reference implementations for architecture study only. |
| Web scraping infra (Firecrawl, Crawl4AI, Scrapy, browser-use) | Low | The infra itself is neutral; risk depends on what target site's ToS says — check before scraping any specific ad network or business site's landing pages beyond ones we're validating with permission. |

## Review gate

No PR that adds a new data source or tool to `modules/` may be merged without:
- [ ] An entry in this table (or a note referencing why it's not needed)
- [ ] A stated legal basis if the source touches personal data
- [ ] A rate-limit / usage-scope note if it queries third-party platforms
