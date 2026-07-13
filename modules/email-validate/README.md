# email-validate

Validation module for the OSINT lead platform. Given a lead record, it answers
the pipeline's `Validate` question for the lead's email — *is it real,
deliverable, low-risk?* — and adds the answer to the record under a namespaced
`email_validate` key, without ever sending an email.

It wraps [AfterShip/email-verifier](https://github.com/AfterShip/email-verifier)
`v1.4.1`, the tool approved for this module in
[`docs/decisions/stage-1-decision.md`](../../docs/decisions/stage-1-decision.md)
and scored 5/5 in
[`evaluations/email-verifier-aftership.md`](../../evaluations/email-verifier-aftership.md).
It implements the module contract in
[`docs/architecture.md`](../../docs/architecture.md).

## Why a stdin/stdout CLI

email-verifier is a Go library, so it has to be exposed to the pipeline somehow.
The two candidates from the evaluation were an embedded call, a long-running
HTTP wrapper (the library ships a reference `cmd/apiserver`), or a CLI filter.
This module is a **CLI that reads one lead as JSON on stdin and writes the
augmented lead as JSON on stdout**, because:

- **No daemon to operate.** A single static binary drops into a sequential /
  DAG module chain — the shape the pipeline is being built from — with no
  service to deploy, health-check, or keep alive. Stage 1 explicitly deferred
  choosing an orchestration backbone until real modules exist; a filter imposes
  the fewest assumptions on whatever runner comes later.
- **Composable and language-agnostic.** Any orchestrator (shell, Python, Go)
  can pipe a record through it. The Go core (`Validator`) is still importable
  directly by a future in-process Go orchestrator, so we keep both options.
- **Clean audit boundary.** The result goes to stdout; the required audit line
  goes to stderr — no interleaving, easy to route to a log sink.

The heavy-weight option (persistent HTTP server) buys connection reuse we don't
need at current volumes and adds operational surface; it can be reintroduced
later by wrapping the same `Validator` type if throughput ever demands it.

## SMTP probe is off by default

The SMTP deliverability probe is **disabled**, matching the evaluation's
recommendation (§6) and the Stage 1 decision. Reasons: most hosts block
outbound port 25 (the probe then hangs to a ~30s timeout), and at scale SMTP
probing can look like address harvesting and trip provider abuse thresholds.
With SMTP off, the module returns syntax / MX / disposable / free / role signals
in single-digit milliseconds. Consequently `deliverable` reports the library's
`reachable` enum, which is `"unknown"` on the non-SMTP path — we do not
fabricate a `true`/`false` we cannot substantiate.

## I/O contract

- **Input (stdin):** a JSON object — a partial lead record with at least an
  `email` field. All other fields are preserved untouched.
- **Output (stdout):** the same object with one key added:

  | field | type | meaning |
  |---|---|---|
  | `status` | string | `"ok"` if the check ran; `"unknown"` on any failure (timeout, tool unavailable, missing email) |
  | `deliverable` | string | mirrors email-verifier `reachable`: `"yes"` / `"no"` / `"unknown"` (always `"unknown"` while SMTP is off) |
  | `syntax_valid` | bool | address is syntactically valid |
  | `has_mx_records` | bool | the domain publishes MX records (can receive mail) |
  | `is_disposable` | bool | disposable/temporary-email domain |
  | `is_role_account` | bool | role address (e.g. `support@`, `info@`) |
  | `is_free_provider` | bool | free-provider domain (e.g. gmail.com) |
  | `checked_at` | string | RFC3339 UTC timestamp of the check |
  | `source_tool` | string | `"AfterShip/email-verifier@v1.4.1"` |
  | `error` | string | present only on failure; human-readable note |

- **Audit (stderr):** exactly one JSON line per call — always, even on failure —
  carrying `tool`, `checked_at`, `email`, `status`, and `legal_basis`
  (`GDPR Art.6(1)(f) legitimate-interest`, per
  [`docs/compliance.md`](../../docs/compliance.md)). This satisfies the
  architecture "Audit" requirement: which tool/version ran, when, and the
  legal-basis tag.

### Failure mode

The module never crashes the pipeline. A missing/empty email, a DNS/MX timeout
(bounded by `EMAIL_VALIDATE_TIMEOUT`, default `10s`), a library panic, or the
tool being otherwise unavailable all produce `status: "unknown"` with an
`error` note — and exit code `0`. A **non-zero exit only** means the input on
stdin was not a readable JSON object.

## Run it

```bash
go build -o email-validate ./cmd/email-validate

echo '{"name":"Jane","email":"support@github.com","company":"Acme"}' \
  | ./email-validate
```

Real output (stdout), from a live run on 2026-07-13:

```json
{
  "company": "Acme",
  "email": "support@github.com",
  "email_validate": {
    "status": "ok",
    "deliverable": "unknown",
    "syntax_valid": true,
    "has_mx_records": true,
    "is_disposable": false,
    "is_role_account": true,
    "is_free_provider": false,
    "checked_at": "2026-07-13T13:45:46Z",
    "source_tool": "AfterShip/email-verifier@v1.4.1"
  },
  "name": "Jane"
}
```

Audit line on stderr for the same call:

```json
{"tool":"AfterShip/email-verifier@v1.4.1","checked_at":"2026-07-13T13:45:46Z","email":"support@github.com","status":"ok","legal_basis":"GDPR Art.6(1)(f) legitimate-interest"}
```

Optional: override the per-call timeout with a Go duration, e.g.
`EMAIL_VALIDATE_TIMEOUT=5s`.

## Test

```bash
go test ./...
```

`TestValidate_RealEmails` runs the real verifier (no mocks) against a valid
well-known address (`support@github.com`) and a malformed one, asserting on the
actual output; `TestValidate_MissingEmail` covers graceful degradation; the CLI
test covers the full stdin→stdout contract and record preservation. The
valid-address case performs a live DNS/MX lookup, so the suite requires outbound
network.

## Dependencies

- Go (built and tested with 1.22.5).
- `github.com/AfterShip/email-verifier v1.4.1` (MIT), pinned in `go.mod`.
- **No API keys, no accounts, no paid services.**
