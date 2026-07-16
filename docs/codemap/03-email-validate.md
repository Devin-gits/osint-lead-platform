# email-validate

**Path:** `modules/email-validate/`  
**Package:** `github.com/Moyeil-73/osint-lead-platform/modules/email-validate`  
**Import alias:** `emailvalidate`  
**Pipeline stage:** Validate  
**Result key:** `email_validate`  
**Decision:** Stage 1 adopt AfterShip email-verifier (5/5); holehe reference-only, not in default path.

## Purpose

Answer: is the lead email real / deliverable / low-risk? **Does not send email.** SMTP probe **disabled**.

## Public API

```go
const SourceTool = "AfterShip/email-verifier@v1.4.1"
const LegalBasis = "GDPR Art.6(1)(f) legitimate-interest"
const DefaultTimeout = 10 * time.Second

func NewValidator(timeout time.Duration) *Validator
func (val *Validator) Validate(email string) (Result, AuditRecord)
```

`NewValidator`: SMTP off, domain suggest on. Non-positive timeout → DefaultTimeout.

## Result fields (`email_validate`)

| Field | Type | Notes |
|-------|------|-------|
| `status` | string | `ok` \| `unknown` |
| `deliverable` | string | `yes` \| `no` \| `unknown` (usually `unknown` with SMTP off) |
| `syntax_valid` | bool | |
| `has_mx_records` | bool | |
| `is_disposable` | bool | |
| `is_role_account` | bool | |
| `is_free_provider` | bool | |
| `checked_at` | string | RFC3339 UTC |
| `source_tool` | string | == SourceTool |
| `error` | string | omitempty; on failure |

## CLI

```bash
cd modules/email-validate
go build -o email-validate ./cmd/email-validate
echo '{"email":"support@github.com"}' | ./email-validate
```

| Env | Default | Meaning |
|-----|---------|---------|
| `EMAIL_VALIDATE_TIMEOUT` | `10s` | Per-call timeout (Go duration) |

## Key implementation details

- Timeout via goroutine + `context.WithTimeout`; library panic recovered
- Missing/empty email → `unknown` + error note
- Audit: one stderr line with raw `email` field

## Tests

```bash
cd modules/email-validate && go test ./...
```

- `TestValidate_RealEmails` — live DNS (network required)
- `TestValidate_MissingEmail` — degrade path
- CLI test — stdin→stdout preservation

## Do not

- Enable SMTP probe without compliance/ops review
- Add holehe to the default automated path
- Change result key name without updating all consumers
