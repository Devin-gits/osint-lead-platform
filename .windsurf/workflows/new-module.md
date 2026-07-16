---
description: Scaffold a Stage 2 OSINT module from the module contract
---

# New module scaffold

Use when implementing an **approved** Stage 2 module under `modules/<name>/`.

## Preconditions

1. Confirm the module is unblocked in `docs/decisions/stage-1-decision.md` and `docs/codemap/08-decisions-and-backlog.md`.
2. Do **not** start `company-enrich` without the local-enrichment-tool second-opinion eval.
3. Read `docs/codemap/01-module-contract.md` and `docs/codemap/09-agent-playbook.md`.
4. Read the relevant evaluation(s) under `evaluations/`.

## Steps

1. Create `modules/<name>/` with its own `go.mod`:
   - module path: `github.com/Moyeil-73/osint-lead-platform/modules/<name>`
   - Go directive: `go 1.22.5` (do not bump unless a decision doc explicitly approves a higher version).

2. Implement library package first (mirror `modules/email-validate/validate.go`):
   - `LegalBasis`, `SourceTool` (or per-source tool constants)
   - `DefaultTimeout`
   - `Result` + `AuditRecord` structs with JSON tags
   - `NewX` constructor + main method that **never returns tool errors** (in-band status)

3. Implement CLI at `cmd/<name>/main.go` (copy pattern from email-validate):
   - Read all stdin → `map[string]interface{}`
   - Call library
   - Write audit JSON line(s) to stderr first
   - Set namespaced key on lead
   - Encode indented JSON to stdout
   - Exit 1 only on bad JSON / read errors

4. Add tests:
   - Package tests for happy path + missing field degrade
   - CLI test for field preservation + audit on stderr
   - Use fakes/httptest for external tools; document if live network is required

5. Write `modules/<name>/README.md` with I/O contract table, env vars, real sample output, compliance notes.

6. Update compliance if new data source: `docs/compliance.md` risk table + legal basis + rate-limit note.

7. Add a codemap page under `docs/codemap/` and link it from `docs/codemap/README.md`.

8. Run tests from the module directory:

```bash
go test ./...
```

## Anti-patterns

- HTTP server as the first integration shape
- Importing GPL tools (theHarvester, PhoneInfoga)
- Overwriting raw lead fields
- Bulk third-party scraping without scope/rate caps
- Inventing fields the underlying tool does not return
