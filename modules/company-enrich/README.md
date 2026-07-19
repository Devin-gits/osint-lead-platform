# company-enrich

Go library and CLI for the `company-enrich` module of the OSINT lead platform.

## Scope

Takes a partial lead record (domain, company name, URL, and optionally prior
`extraction`/`domain_intel` results) and enriches it with public, company-level
firmographics.

**This module does NOT:**

- Scrape LinkedIn or Crunchbase.
- Use LLMs (Anthropic/OpenAI/etc.) to synthesize company facts.
- Return people/contact data such as CEO, founders, or job postings.
- Make paid API calls unless an API key is explicitly configured.

## Architecture

```
Input
  │
  ▼
Enricher
  │
  ├─► local provider (always, no API key)
  │     • domain / company / URL normalization
  │     • reuse extraction.company_name / description / social_links
  │     • optional GitHub public org API lookup (only when extraction social_links
  │       contain a GitHub org URL, or when COMPANY_ENRICH_GITHUB_DOMAIN_GUESS=1)
  │
  └─► discolike provider (only when DISCOLIKE_API_KEY set)
        • thin Go HTTP client to api.discolike.com/v1/profile (+ vendors)

  ▼
merge (gap-fill only) → result JSON + stderr audit lines
```

The waterfall short-circuits once the requested `required_fields` are satisfied.
By default the required set is the P0 fields: `domain`, `name`, `website`.

## Status semantics

| Status | Meaning |
|---|---|
| `ok` | All P0 fields (`domain`, `name`, `website`) are present after merge. |
| `partial` | At least one useful field beyond the lookup key was found, but P0 is incomplete. |
| `skipped` | `permission_ref` missing, or all of `domain`/`company`/`url` missing. |
| `error` | Providers were attempted and all hard-failed with no usable data. |

## I/O contract

### Input (stdin for CLI)

```json
{
  "domain": "example.com",
  "company": "Example",
  "url": "https://example.com",
  "permission_ref": "CAMP-2026-Q3-001",
  "extraction": {
    "status": "ok",
    "fields": {
      "company_name": "Example, Inc.",
      "description": "Enterprise widgets.",
      "social_links": ["https://github.com/example"]
    }
  },
  "required_fields": ["domain", "name", "website"]
}
```

`permission_ref` is mandatory. At least one of `domain`, `company`, or `url` must be provided.

### Output (stdout for CLI)

```json
{
  "status": "ok",
  "source_tool": "company-enrich/local",
  "confidence": 0.6,
  "fields": {
    "domain": "example.com",
    "name": "Example, Inc.",
    "website": "https://example.com",
    "description": "Enterprise widgets.",
    "social_links": {
      "github": "https://github.com/example"
    },
    "sources": ["local"]
  },
  "metadata": {
    "backend": "company-enrich",
    "legal_basis": "GDPR Art.6(1)(f) legitimate-interest",
    "permission_ref": "CAMP-2026-Q3-001",
    "duration_ms": 12,
    "limits_applied": "timeout=30s,min_interval=500ms,providers=local,discolike"
  },
  "checked_at": "2026-07-19T20:00:00Z"
}
```

### Stderr audit

One JSON line per provider invocation plus a final module audit line:

```json
{
  "module": "company-enrich",
  "tool": "local",
  "tool_version": "company-enrich/local",
  "timestamp": "2026-07-19T20:00:00Z",
  "legal_basis": "GDPR Art.6(1)(f) legitimate-interest",
  "permission_ref": "CAMP-2026-Q3-001",
  "subject": {"domain": "example.com", "company": "Example"},
  "status": "ok",
  "duration_ms": 0,
  "limits": "timeout=30s,min_interval=500ms,providers=local,discolike"
}
```

No personal names or emails appear in audit records.

## Configuration / env vars

| Variable | Default | Purpose |
|---|---|---|
| `DISCOLIKE_API_KEY` | empty | Activates the DiscoLike adapter. If unset, the adapter returns `skipped`. |
| `DISCOLIKE_BASE_URL` | `https://api.discolike.com/v1` | Override DiscoLike API base URL (tests, on-prem). |
| `DISCOLIKE_TIMEOUT` | `30s` | Per-request timeout. |
| `COMPANY_ENRICH_GITHUB_DOMAIN_GUESS` | `0` | When `1`/`true`/`yes`, the local provider will weakly guess a GitHub org from the domain root. Disabled by default. |

## Running the CLI

```bash
go build -o /tmp/company-enrich ./cmd/company-enrich

echo '{"domain":"example.com","company":"Example","permission_ref":"DEMO-1"}' | /tmp/company-enrich

# With extraction context
cat <<'EOF' | /tmp/company-enrich
{
  "domain": "stripe.com",
  "permission_ref": "DEMO-1",
  "extraction": {
    "status": "ok",
    "fields": {
      "company_name": "Stripe, Inc.",
      "description": "Payments infrastructure for the internet.",
      "social_links": ["https://github.com/stripe"]
    }
  }
}
EOF
```

Exit code is `0` for all operational outcomes (`ok`/`partial`/`skipped`/`error`).
Exit code is non-zero only for unreadable/malformed input.

## Testing

```bash
go test ./...
go test ./... -short   # also passes; no live API calls required
go vet ./...
go build -o /tmp/company-enrich ./cmd/company-enrich
```

## Confidence formula

The module computes a conservative 0.0–1.0 score based on populated field
categories:

| Field | Weight |
|---|---|
| `domain` | 0.10 |
| `name` | 0.20 |
| `website` | 0.10 |
| `description` | 0.15 |
| `legal_name` | 0.10 |
| `founded` | 0.10 |
| `employee_count` / `employee_count_range` | 0.10 |
| `headquarters.country` | 0.10 |
| `industry` (any) | 0.05 |
| `tech_stack` (any) | 0.05 |

Score is capped at 1.0.

## License

MIT. This module does not import or link any GPL/AGPL code. The optional
DiscoLike adapter is a small Go HTTP client that calls a paid API; no DiscoLike
or local-enrichment-tool source is vendored.
