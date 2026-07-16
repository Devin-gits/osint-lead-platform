# domain-intel

**Path:** `modules/domain-intel/`  
**Package:** `github.com/Moyeil-73/osint-lead-platform/modules/domain-intel`  
**Import alias:** `domainintel`  
**Pipeline stage:** Ingest  
**Result key:** `domain_intel`  
**Decision:** Run **both** web-check signals + theHarvester (time-boxed on keyless value).

## Purpose

1. **web_check:** is this an established, real business domain? (DNS, SSL, WHOIS age)  
2. **harvester:** what hosts/subdomains/emails hang off it?

## Public API

```go
const LegalBasis = "GDPR Art.6(1)(f) legitimate-interest"
const DefaultTimeout = 60 * time.Second  // per sub-tool

func NewAnalyzer(timeout time.Duration) *Analyzer
func (a *Analyzer) Analyze(domain string) (Result, []AuditRecord)  // always 2 audits
```

Sub-tools run **concurrently** (`sync.WaitGroup`); each has panic-recover wrappers (`safeWebCheck`, `safeHarvester`).

## Result shape

```
domain_intel:
  web_check:
    status, resolvable, dns{a,aaaa,mx,ns,txt}, ssl{...}, whois{...},
    checked_at, source_tool, error?
  harvester:
    status, hosts[{host,ip?}], host_count, ips[], emails[], sources[],
    checked_at, source_tool, error?
  checked_at
  source_tools[]
```

### web-check-lite (`webcheck.go`)

- **Not** a fork of lissy93/web-check Node app — reimplements DNS/SSL/WHOIS in Go stdlib
- `status: ok` if DNS resolves (A or AAAA); SSL/WHOIS best-effort
- Tool id: `web-check-lite (reimpl. of lissy93/web-check@2.1.10 dns/ssl/whois checks)`

### theHarvester (`harvester.go`)

- **CLI subprocess only** — never import Python (GPL-2.0 license firewall)
- Tool id: `laramies/theHarvester@v4.11.1 (CLI subprocess)`
- Fixed allowlist `-b`: `hackertarget,crtsh,rapiddns,certspotter`
- **Excluded:** breach DBs (haveibeenpwned, dehashed, leaklookup), paid sources
- Parses `-f` JSON (`hosts`, `ips`, `emails`); hosts as `subdomain:ip` strings
- Missing binary → `unknown` + install hint; does not block web_check

## CLI

```bash
cd modules/domain-intel
go build -o domain-intel ./cmd/domain-intel
echo '{"domain":"owasp.org"}' | ./domain-intel
```

| Env | Default | Meaning |
|-----|---------|---------|
| `DOMAIN_INTEL_TIMEOUT` | `60s` | Per sub-tool timeout |
| `DOMAIN_INTEL_HARVESTER_BIN` | `theHarvester` | Binary path/name |

Domain input: bare domain or URL; `normalizeDomain` strips scheme/path/port.

## Tests

```bash
cd modules/domain-intel
go test ./...           # needs network; full harvester if installed
go test -short ./...    # skips live network/subprocess
```

Notable: `TestAllowlistExcludesBreachDBs`, `TestHarvesterAbsent`, `TestAnalyze_RealDomain`.

## Downstream consumers

`social-footprint` may read `domain_intel.harvester` from an already-enriched lead for secondary handle candidates (`handles.go` → `harvesterHandles`).

## Do not

- Import theHarvester as a library
- Widen `-b` allowlist without compliance review
- Require theHarvester for the module to “work” (optional at runtime)
