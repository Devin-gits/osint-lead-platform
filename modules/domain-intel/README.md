# domain-intel

Ingest-stage module for the OSINT lead platform. Given a lead record, it answers
the pipeline's **Ingest** question for the lead's `domain` — *is this an
established, real business domain, and what hosts/contacts hang off it?* — and
adds the answer to the record under a namespaced `domain_intel` key, without
overwriting any raw ingested field.

Per the Stage 1 decision
([`docs/decisions/stage-1-decision.md` → `domain-intel`](../../docs/decisions/stage-1-decision.md))
this module runs **both** approved tools, because they answer different
questions:

| Sub-tool | Question it answers | Integration style |
|---|---|---|
| **web-check** ([eval, 5/5](../../evaluations/web-check.md)) | "Is this an established, real business domain?" — DNS, SSL, WHOIS/domain-age | Lightweight reimplementation (Go stdlib) |
| **theHarvester** ([eval, 4/5](../../evaluations/theharvester.md)) | "What hosts/subdomains/contacts hang off this domain?" | External CLI subprocess only |

It implements the module contract in
[`docs/architecture.md`](../../docs/architecture.md): partial-record in,
same-record-plus-namespaced-key out, graceful degradation to `unknown`, and a
per-tool audit log.

## Language: Go

The module's own wrapper code is **Go**, matching the sibling
[`email-validate`](../email-validate) module (same repo module path convention,
same stdin/stdout CLI shape) so the pipeline is built from one consistent
binary-per-module toolchain. Go also fits this module's two integration jobs
cleanly with one permissive Go dependency: `os/exec` drives the theHarvester
subprocess, `net` / `net/http` / `crypto/tls` cover DNS, HTTP and TLS, and the
Apache-2.0 `github.com/likexian/whois` client handles WHOIS referrals.
theHarvester itself is Python,
but that is irrelevant to the wrapper language — it is invoked as an external
process, never imported (see below).

## The two integrations, and why they differ

### theHarvester — CLI subprocess only (never a library import)

theHarvester is **GPL-2.0**; this repo's code is **MIT**. Importing its Python
modules into our code and distributing the result would trigger GPL-2.0's
copyleft over the combined work. Per its
[evaluation §2/§8](../../evaluations/theharvester.md) and the Stage 1 decision,
we therefore invoke it **only as a separate CLI subprocess** — *mere
aggregation*, which imposes no copyleft obligation. Concretely, `harvester.go`
shells out to:

```
theHarvester -d <domain> -b hackertarget,crtsh,rapiddns,certspotter -l 200 -f <tmp>/out
```

and parses the `<tmp>/out.json` file theHarvester writes. The parsing is based
on theHarvester v4.11.1's **actual** `-f` JSON output — top-level keys `cmd`,
`hosts` (each a `"<subdomain>:<ip>"` string), `ips`, `emails`, `shodan`, plus
optional keys — verified against the installed source (`theHarvester/__main__.py`
JSON report section) and a real run (see [Testing](#testing)). An offline fake
subprocess covers argv enforcement and parsing without replacing live integration tests.

**Source allowlist (compliance).** The `-b` list is a fixed constant
(`hackertarget`, `crtsh`, `rapiddns`, `certspotter`) — all **keyless** and
**non-breach-database**. Breach-database modules (`haveibeenpwned`, `dehashed`,
`leaklookup`) are **deliberately excluded** per the Stage 1 decision's
compliance note and [`docs/compliance.md`](../../docs/compliance.md) hard-rule
#3; paid/keyed sources (Hunter, SecurityTrails, Shodan, Censys) are excluded too
since keyless operation is the documented default. Tests
(`TestAllowlistExcludesBreachDBs` and `TestHarvesterArgvExcludesBlockedSources`)
enforce the exclusion in configuration and on the actual subprocess argv.

**If theHarvester isn't installed**, the harvester sub-result degrades to
`status: "unknown"` with an install hint — it never blocks web-check or crashes
the pipeline. Install it separately (it is **not** on PyPI as the real tool —
that name is a v0.0.1 stub; use source/`uv`/Docker):

```bash
# from source (per theHarvester README)
git clone https://github.com/laramies/theHarvester
cd theHarvester && pip install .        # or: uv sync && uv run theHarvester
# then ensure `theHarvester` is on PATH, or point the module at it:
export DOMAIN_INTEL_HARVESTER_BIN=/path/to/theHarvester
```

### web-check — lightweight reimplementation, not a fork of the full app

web-check's evaluation recommended "fork & modify", but web-check is a full
Node/React web app with ~35 checks. Forking and operating the entire app (Node
runtime, optional chromium/traceroute binaries, a service to deploy) is
disproportionate for the handful of signals the Ingest stage actually needs, and
it would not compose as a single static binary the way the pipeline's other
modules do. So this module **reimplements the specific web-check checks relevant
to "is this an established, real business domain"** — the exact subset the
evaluation itself named (DNS, SSL/TLS, WHOIS/domain-age) — directly in Go stdlib:

- **DNS** (`api/dns` equivalent): A / AAAA / CNAME / MX / NS / TXT via `net.Resolver`.
- **SSL/TLS** (`api/ssl` equivalent): dials `:443`, reads the leaf certificate's
  issuer/subject/validity window/SANs and negotiated protocol via `crypto/tls`.
- **HTTP**: requests HTTPS first and falls back to HTTP, recording status,
  `Server`, `Content-Type`, and `X-Powered-By` with a bounded `net/http` client.
- **WHOIS + domain age** (`api/whois` equivalent): a two-step WHOIS query over
  raw TCP/43 (ask `whois.iana.org` for the TLD's authoritative server, then
  query that server), parsing the creation date and registrar and deriving
  `domain_age_days`.

WHOIS uses `github.com/likexian/whois` **v1.15.6** (Apache-2.0). This version
declares Go 1.21 and therefore builds without changing the module's exact Go
1.22.5 pin; v1.15.7 was not selected because it requires Go 1.24 or newer.

The output field shapes mirror web-check's real JSON (e.g. its `api/dns.js`
returns `A`/`AAAA`/`CNAME`/`MX`/`NS`/`TXT`, restated here in lower-case JSON keys). If a
future need arises for web-check's richer checks (tech-stack, blocklist,
headers), the fork option remains open behind this same interface; the
reimplementation is the pragmatic Stage 2 starting point, not a permanent
rejection of the fork.

## I/O contract

- **Input (stdin):** a JSON object — a partial lead record with at least a
  `domain` field. A bare domain or a full URL is accepted; scheme/path/port are
  stripped. All other fields are preserved untouched.
- **Output (stdout):** the same object with one key, `domain_intel`, added:

  ```
  domain_intel:
    web_check:                    # establishment/legitimacy signals
      status            ok|unknown  ("ok" once DNS resolves)
      resolvable        bool
      dns               {a[], aaaa[], cname[], mx[], ns[], txt[]}
      ssl               {valid, issuer, subject, not_before, not_after, days_until_expiry, protocol?, sans[]?, error?}
      http              {status_code, server?, headers?, error?}
      whois             {registrar, created_date, domain_age_days, error?}
      checked_at        RFC3339 UTC
      source_tool       string
      error             string   (present only on total failure)
    harvester:                    # host/subdomain/contact inventory
      status            ok|unknown
      hosts             [{host, ip?}]   (parsed from theHarvester "subdomain:ip")
      host_count        int
      ips               [string]
      emails            [string]
      sources           [string]  (the allowlisted -b sources queried)
      checked_at        RFC3339 UTC
      source_tool       string
      error             string   (present only on failure)
    checked_at          RFC3339 UTC
    source_tools        [web_check tool, harvester tool]
  ```

- **Audit (stderr):** exactly **two** JSON lines per call — **one per tool**,
  always, even on failure — each carrying `tool` (name + version), `checked_at`,
  `domain`, `status`, `legal_basis`, and an optional `error` note
  (`GDPR Art.6(1)(f) legitimate-interest`, per
  [`docs/compliance.md`](../../docs/compliance.md), where domain intel is the
  "Low" personal-data-risk category). This satisfies the architecture "Audit"
  requirement for **both** tools: which tool/version ran, when, and the
  legal-basis tag.

### Failure mode

The module never crashes the pipeline. The two sub-tools run **concurrently and
independently**: if theHarvester times out, is missing, or crashes, its
sub-result degrades to `status: "unknown"` with an `error` note while web-check
still returns its result — and vice-versa. Each sub-tool is bounded by
`DOMAIN_INTEL_TIMEOUT` (a Go duration, default `60s`) applied **per tool**, and
each is wrapped in a panic-recover. Exit code is `0` even on sub-tool failure; a
**non-zero exit only** means stdin was not a readable JSON object.

## Run it

```bash
go build -o domain-intel ./cmd/domain-intel

echo '{"company":"OWASP Foundation","domain":"owasp.org","email":"info@owasp.org"}' \
  | ./domain-intel
```

Real output (stdout), from a live run on 2026-07-13 with theHarvester v4.11.1
installed (host list trimmed to representative rows — the run returned 147):

```json
{
  "company": "OWASP Foundation",
  "domain": "owasp.org",
  "email": "info@owasp.org",
  "domain_intel": {
    "web_check": {
      "status": "ok",
      "resolvable": true,
      "dns": {
        "a": ["172.66.157.115", "104.20.44.163"],
        "aaaa": ["2606:4700:10::6814:2ca3", "2606:4700:10::ac42:9d73"],
        "cname": [],
        "mx": ["aspmx.l.google.com", "alt2.aspmx.l.google.com", "alt1.aspmx.l.google.com", "alt4.aspmx.l.google.com", "alt3.aspmx.l.google.com"],
        "ns": ["fay.ns.cloudflare.com", "west.ns.cloudflare.com"],
        "txt": ["v=spf1 include:_spf.google.com include:servers.mcsv.net include:amazonses.com -all", "MS=ms73859685", "..."]
      },
      "ssl": {
        "valid": true,
        "issuer": "CN=WE1,O=Google Trust Services,C=US",
        "subject": "CN=owasp.org",
        "not_before": "2026-07-08T04:07:11Z",
        "not_after": "2026-10-06T05:06:52Z",
        "days_until_expiry": 84,
        "protocol": "TLS 1.3",
        "sans": ["owasp.org", "*.owasp.org"]
      },
      "http": {
        "status_code": 200,
        "server": "cloudflare",
        "headers": {"Content-Type": "text/html; charset=utf-8"}
      },
      "whois": {
        "registrar": "GoDaddy.com, LLC",
        "created_date": "2001-09-21T17:00:36Z",
        "domain_age_days": 9060
      },
      "checked_at": "2026-07-13T13:59:51Z",
      "source_tool": "web-check-lite (reimpl. of lissy93/web-check@2.1.10 dns/tls/http/whois checks)"
    },
    "harvester": {
      "status": "ok",
      "hosts": [
        {"host": "*.owasp.org"},
        {"host": "20thanniversary.owasp.org", "ip": "172.66.157.115"},
        {"host": "20thanniversary.owasp.org", "ip": "2606:4700:10::6814:2ca3"},
        {"host": "mail.owasp.org", "ip": "104.20.44.163"},
        {"host": "wiki.owasp.org", "ip": "172.66.157.115"}
      ],
      "host_count": 147,
      "ips": [],
      "emails": [],
      "sources": ["hackertarget", "crtsh", "rapiddns", "certspotter"],
      "checked_at": "2026-07-13T13:59:51Z",
      "source_tool": "laramies/theHarvester@v4.11.1 (CLI subprocess)"
    },
    "checked_at": "2026-07-13T13:59:51Z",
    "source_tools": [
      "web-check-lite (reimpl. of lissy93/web-check@2.1.10 dns/tls/http/whois checks)",
      "laramies/theHarvester@v4.11.1 (CLI subprocess)"
    ]
  }
}
```

Audit lines on stderr for the same call (one per tool):

```json
{"tool":"web-check-lite (reimpl. of lissy93/web-check@2.1.10 dns/tls/http/whois checks)","checked_at":"2026-07-13T13:59:51Z","domain":"owasp.org","status":"ok","legal_basis":"GDPR Art.6(1)(f) legitimate-interest"}
{"tool":"laramies/theHarvester@v4.11.1 (CLI subprocess)","checked_at":"2026-07-13T13:59:51Z","domain":"owasp.org","status":"ok","legal_basis":"GDPR Art.6(1)(f) legitimate-interest"}
```

### Graceful degradation (theHarvester unavailable)

Same command with theHarvester not on PATH — web-check still returns `ok`, the
harvester sub-result degrades to `unknown`, and both audit lines are still
emitted (captured live 2026-07-13):

```json
"harvester": {
  "status": "unknown",
  "hosts": [],
  "host_count": 0,
  "ips": [],
  "emails": [],
  "sources": ["hackertarget", "crtsh", "rapiddns", "certspotter"],
  "checked_at": "2026-07-13T14:00:15Z",
  "source_tool": "laramies/theHarvester@v4.11.1 (CLI subprocess)",
  "error": "theHarvester not found (\"/nonexistent/theHarvester\"); install it separately and/or set DOMAIN_INTEL_HARVESTER_BIN — see README"
}
```

## Configuration

| Env var | Default | Meaning |
|---|---|---|
| `DOMAIN_INTEL_TIMEOUT` | `60s` | Per-sub-tool timeout (Go duration, e.g. `30s`). |
| `DOMAIN_INTEL_HARVESTER_BIN` | `theHarvester` (on PATH) | Path/name of the theHarvester executable. |

## Test

```bash
go test ./...           # requires outbound network (+ theHarvester for full coverage)
go test -short ./...    # skips the live network/subprocess tests
```

- `TestAnalyze_RealDomain` runs **both real sub-tools** (no mocks) against
  `owasp.org`: asserts web-check resolves A records and a valid SSL cert, and —
  if theHarvester is installed — that it returns hosts; if it is not installed,
  it asserts the graceful degrade to `unknown` instead. Live DNS/SSL/WHOIS +
  subprocess, so it needs network.
- `TestAnalyze_MissingDomain` covers graceful degradation for an empty domain
  (both tools `unknown`, audit lines still emitted).
- `TestHarvesterAbsent` forces the missing-binary path (no network).
- `TestHarvesterArgvExcludesBlockedSources` runs an offline fake executable,
  verifies `-l`/`-f`, proves breach sources never reach argv, and parses JSON.
- `TestNormalizeDomain`, `TestSplitHost` (incl. IPv6), `TestInspectHTTP`,
  `TestResultJSONSchema`, and `TestTLSVersionName` cover pure/offline invariants.
- The CLI test (`cmd/domain-intel`) covers the full stdin→stdout contract,
  raw-field preservation, and the two-audit-lines-on-stderr behavior.

## Dependencies

- Go (built and tested with 1.22.5), plus `github.com/likexian/whois` v1.15.6
  (Apache-2.0) for bounded WHOIS/referral queries. Native DNS/TLS/HTTP and CLI
  integration use `net`, `crypto/tls`, `net/http`, `os/exec`, and `encoding/json`.
- **theHarvester v4.11.1** — *external, optional at runtime*, invoked as a CLI
  subprocess (never imported). Install separately (source/`uv`/Docker; not the
  PyPI stub). GPL-2.0; kept behind the subprocess boundary so this MIT module
  incurs no copyleft obligation. Absent → harvester sub-result is `unknown`.
- **No API keys, no accounts, no paid services** — keyless theHarvester sources
  and keyless DNS/TLS/HTTP/WHOIS only.

The Stage 1 approval is time-boxed: after deployment, re-assess whether the
keyless-only theHarvester results add enough enrichment value to keep this
integration automated; otherwise demote it to an optional/manual module.
