# Agent playbook

How to change this codebase without breaking contracts or compliance.

## Before any code change

1. Read the relevant module codemap + `docs/compliance.md` if touching data sources.
2. Confirm Stage 1 decision allows the work (`08-decisions-and-backlog.md`).
3. Prefer **minimal diffs** matching existing style (long package docs, in-band status, no mid-file imports).

## Adding or extending a Stage 2 module

Copy structure from **email-validate** (simplest) or **domain-intel** (multi-source):

```
modules/<name>/
  go.mod          # github.com/Moyeil-73/osint-lead-platform/modules/<name>
  <pkg>.go        # NewX, Result, AuditRecord, LegalBasis, SourceTool
  cmd/<name>/main.go
  *_test.go
  README.md       # I/O contract table + real sample output
```

### Must implement

- [ ] stdin JSON → stdout JSON + namespaced key
- [ ] stderr audit JSON (tool, checked_at, subject, status, legal_basis)
- [ ] exit 0 on tool failure; non-zero only on bad JSON
- [ ] graceful `unknown` / `skipped`
- [ ] timeout + panic recover
- [ ] package tests + CLI contract test
- [ ] compliance table / legal basis if new source

### Must not

- [ ] HTTP server as default (CLI filter first)
- [ ] Overwrite raw lead fields
- [ ] Import GPL tools
- [ ] Invent result fields that tools don't actually return
- [ ] Bulk third-party scraping without rate/scope caps
- [ ] Open `company-enrich` or orchestration without decision update

## Multi-source modules

Pattern: concurrent goroutines + independent degrade + one audit per source + merge for top-level verdict. See `domainintel.Analyze`, `phonevalidate.Validate`.

## Subprocess pattern

- Argv slice only (no shell)
- Fixed allowlists as constants (not from lead input)
- Temp dirs cleaned with `defer`
- Missing binary → unknown + install hint

## Testing conventions

| Flag / style | Use |
|--------------|-----|
| `go test ./...` | default |
| `go test -short ./...` | domain-intel skips live net |
| Fake interfaces | social-footprint `maigretRunner` |
| httptest | phone-validate numverify |
| Live network | email MX, domain DNS — document in README |

## Manual smoke

```bash
cd modules/<name>
go build -o <name> ./cmd/<name>
echo '{...}' | ./<name> 2>audit.ndjson | jq .
```

## Research (Stage 1) PRs

- Touch **only** `evaluations/<tool-slug>.md` from TEMPLATE
- Every claim linked; license accurate; fit score vs pipeline
- Title: `research: evaluate {TOOL}`

## Implementation PR titles (historical)

`feat: implement modules/<name>`

## When uncertain

- Prefer reading module README + tests over inventing behavior
- Prefer `unknown` over crashing
- Prefer documenting license/compliance risk over shipping gray tools
