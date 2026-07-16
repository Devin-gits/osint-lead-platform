# Compliance and license

Primary sources: `docs/compliance.md`, Stage 1 decision, module READMEs.

## Hard rules (never violate in default pipeline)

1. **No non-consensual personal surveillance** — reverse-image, private social graph, location history out of scope without legal sign-off.
2. **Respect third-party ToS** — LinkedIn profile scraping **excluded from production**.
3. **Rate-limit gray-zone lookups** — holehe, h8mail, GHunt-style: spot-check flagged leads only, documented Art. 6(1)(f), not bulk default.
4. **Log legal basis** (and ideally permission_ref) every enrichment/validation run.
5. **Retention** — define windows; failed leads shorter than converted (Stage 3; not implemented yet).

## Risk table (summary)

| Category | Risk | Platform stance |
|----------|------|-----------------|
| Domain intel (web-check, theHarvester) | Low | Implemented; keyless non-breach sources only |
| Email verify (AfterShip) | Low | Implemented; no third-party PII send |
| Email→account (holehe) | Medium | **Not** default path |
| Breach (h8mail) | Medium-High | **Rejected** as pipeline module |
| Phone OSINT | Low-Medium | Implemented; optional numverify |
| Social footprint | Medium | Implemented with hard scope/rate caps |
| Reverse-image / GHunt | High | Excluded |
| LinkedIn scrape | High | Excluded |
| Crawl infra (Crawl4AI/Firecrawl) | Low (target-dependent) | Approved for extraction; not built |

## Legal basis constant

Modules log:

```
GDPR Art.6(1)(f) legitimate-interest
```

Business permission ≠ individual consent — each technique still needs its own basis check.

## License firewalls (critical for agents)

| Tool | License | Integration rule |
|------|---------|------------------|
| Platform code | MIT | — |
| AfterShip email-verifier | MIT | Import OK |
| nyaruka/phonenumbers | MIT | Import OK |
| Maigret | MIT | Import in Python wrapper OK |
| theHarvester | **GPL-2.0** | **Subprocess only** — never import |
| PhoneInfoga | **GPL-3.0** | **Do not import**; use libphonenumber directly |
| Firecrawl | AGPL-3.0 | Hosted/self-host tension; not permanent default without plan |
| Crawl4AI | Apache-2.0 + attribution clause | Preferred extraction engine |

**Mere aggregation** (CLI subprocess) keeps MIT code clear of GPL copyleft. Library import + distribution does not.

## PR review gate (from compliance.md)

No PR adding a data source to `modules/` without:

- [ ] Risk table entry (or note why N/A)
- [ ] Stated legal basis if personal data
- [ ] Rate-limit / usage-scope note if third-party platforms queried

## PII in logs

| Module | Audit subject | Redaction |
|--------|---------------|-----------|
| email-validate | email | none (full email) |
| domain-intel | domain | n/a |
| phone-validate | phone | **redacted** |
| social-footprint | handle | never email/name |

## Explicitly rejected / blocked

- h8mail as pipeline module
- SpiderFoot as orchestrator (for now)
- holehe in automated per-lead path
- company-enrich code until local-enrichment-tool second-opinion eval
