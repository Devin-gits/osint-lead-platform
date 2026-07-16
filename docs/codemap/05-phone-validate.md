# phone-validate

**Path:** `modules/phone-validate/`  
**Package:** `github.com/Moyeil-73/osint-lead-platform/modules/phone-validate`  
**Import alias:** `phonevalidate`  
**Pipeline stage:** Validate  
**Result key:** `phone_validate`  
**Decision:** Offline local scanner now; numverify optional/swappable. **No PhoneInfoga runtime dep** (GPL-3.0, unmaintained).

## Purpose

Is the phone well-formed/valid, and what country / line-type / carrier?

## Public API

```go
const LegalBasis = "GDPR Art.6(1)(f) legitimate-interest"
const DefaultTimeout = 10 * time.Second  // per source

func NewValidator(timeout time.Duration) *Validator  // numverify from env
func (v *Validator) Validate(phone string) (Result, []AuditRecord)  // always 2 audits
```

Local + numverify run **concurrently**. Merge: numverify wins for `line_type`/`carrier`/`country` when status `ok`.

## Result fields (top-level merged)

| Field | Notes |
|-------|-------|
| `status` | `ok` if local parsed; else `unknown` |
| `format_valid` | plausible shape |
| `is_valid_number` | libphonenumber assignable |
| `line_type` | mobile, fixed_line, voip, … or `unknown` |
| `carrier` | string or `unknown` |
| `country` | ISO alpha-2 or `unknown` |
| `e164`, `national`, `country_code` | when parseable |
| `local` | full offline sub-result |
| `numverify` | ok \| skipped \| unknown |
| `checked_at`, `source_tools` | |

### local (`local.go`)

- Engine: `github.com/nyaruka/phonenumbers` v1.5.0 (MIT)
- Expects international/E.164-ish input (`+` country code)
- Offline carrier via `carrier` subpackage (often empty for US portability)

### numverify (`numverify.go`)

| Status | When |
|--------|------|
| `skipped` | `NUMVERIFY_API_KEY` unset — **not a failure** |
| `ok` | API success |
| `unknown` | network/HTTP/API error envelope |

Constants: `APIKeyEnv`, `BaseURLEnv`, default base `https://apilayer.net/api/validate`.

## CLI

```bash
cd modules/phone-validate
go build -o phone-validate ./cmd/phone-validate
echo '{"phone":"+14152007986"}' | ./phone-validate
```

| Env | Default | Meaning |
|-----|---------|---------|
| `PHONE_VALIDATE_TIMEOUT` | `10s` | Per-source timeout |
| `NUMVERIFY_API_KEY` | unset | Enables numverify |
| `NUMVERIFY_BASE_URL` | apilayer validate URL | Override / test stub |

## Audit PII

`phone` on audit lines is **redacted** (e.g. `+14*******86`). See `redact` in package.

## Tests

```bash
cd modules/phone-validate && go test ./...
```

- Real offline numbers (no network)
- httptest stub for numverify success + API error degrade
- `TestRedact`, CLI contract tests

## Go version

`go 1.22.5` (phonenumbers v1.5.0 is fully compatible).

## Do not

- Import PhoneInfoga (GPL-3.0)
- Treat missing API key as hard failure
- Log raw phone numbers in audit
