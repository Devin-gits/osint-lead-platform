# phone-validate

Validation-stage module for the OSINT lead platform. Given a lead record, it
answers the pipeline's **Validate** question for the lead's `phone` — *is this a
real, well-formed number, and what country / line-type / carrier does it map
to?* — and adds the answer to the record under a namespaced `phone_validate`
key, without overwriting any raw ingested field.

Per the Stage 1 decision
([`docs/decisions/stage-1-decision.md` → `phone-validate`](../../docs/decisions/stage-1-decision.md),
a reviewer judgment call, scored 3/5 in
[`evaluations/phoneinfoga.md`](../../evaluations/phoneinfoga.md)) the value
splits cleanly in two, and this module is built accordingly:

| Sub-source | Question it answers | Integration style | Requires a key? |
|---|---|---|---|
| **local scanner** | validity, format/normalization, country, line-type, offline carrier | libphonenumber engine, imported directly | No |
| **numverify** | live carrier / line-type lookup | optional HTTP API behind `NUMVERIFY_API_KEY` | Yes (optional) |

It implements the module contract in
[`docs/architecture.md`](../../docs/architecture.md): partial-record in,
same-record-plus-namespaced-key out, graceful degradation to `unknown`, and a
per-source audit log.

## PhoneInfoga maintenance status — checked live 2026-07-13

The Stage 1 decision required re-verifying PhoneInfoga's repo status *before
building*, because its evaluation flagged it as possibly archived. **Checked live
against the GitHub API on 2026-07-13:**

- `sundowndev/phoneinfoga` is **NOT archived** and **not disabled** (`archived:
  false`, `disabled: false`).
- Last push: **2026-01-06** — but per its evaluation that latest commit is a
  **docs-only** README status update; the last functional release is **v2.11.0
  (2024-02-21)**, ~2.5 years stale.
- Its own README still declares the project **"stable but unmaintained… bugs
  won't be fixed and repository could be archived at any time."**
- License: **GPL-3.0** (strong copyleft).

**Maintenance-risk conclusion:** the repo is alive *today* but self-declared
unmaintained and archive-at-any-time. That is exactly why this module does **not
take a runtime dependency on PhoneInfoga**. Instead it builds the local scanner
directly on the same engine PhoneInfoga's `local` scanner wraps —
[`github.com/nyaruka/phonenumbers`](https://github.com/nyaruka/phonenumbers), the
**MIT-licensed, actively-maintained** Go port of Google's libphonenumber (latest
`v2.0.3`, pushed 2026-07-10). This is precisely the substitution the decision doc
anticipated ("evaluate whether a maintained phone-parsing library — Google's
libphonenumber — can replace the local scanner entirely, reducing our exposure to
the archived repo"). If PhoneInfoga is archived tomorrow, this module is
unaffected.

## Why build on the engine directly instead of importing PhoneInfoga

PhoneInfoga's `local` scanner is a thin wrapper: it calls
`phonenumbers.Parse` / `IsValidNumber` / `Format` / `GetRegionCodeForNumber`
from `github.com/nyaruka/phonenumbers` and returns the parsed fields
([`lib/number/number.go`](https://github.com/sundowndev/phoneinfoga/blob/master/lib/number/number.go)).
There is no proprietary logic to lose by not importing it.

Two independent reasons make direct import of PhoneInfoga the wrong choice, and
using its underlying engine the right one:

1. **License boundary.** PhoneInfoga is **GPL-3.0**; this repo is **MIT**.
   Importing its Go module and distributing the result would pull the combined
   work under GPL-3.0 — the same copyleft trap the sibling
   [`domain-intel`](../domain-intel) module avoids for GPL-2.0 theHarvester (by
   subprocess-only invocation). `nyaruka/phonenumbers` is **MIT**, so importing
   it directly keeps this module cleanly MIT.
2. **Maintenance risk.** See above — depending on an archive-at-any-time repo for
   a production validator is unacceptable; the maintained engine is not.

The result is the *same offline signal PhoneInfoga's local scanner produces*
(and slightly more: PhoneInfoga's local output does not surface line-type or a
resolved carrier name, whereas the engine's `GetNumberType` and `carrier`
subpackage do), with none of the license or maintenance exposure. This mirrors
`domain-intel`'s "reimplement the specific checks against a clean dependency
rather than fork the whole GPL app" decision.

## The optional numverify path

`numverify` (an APILayer product) is a live carrier/line-type lookup API. Per the
Stage 1 decision it is a **thin, optional, swappable dependency — never required
for the module to function**:

- **No key configured** (`NUMVERIFY_API_KEY` unset/empty) → the `numverify`
  sub-result reports `status: "skipped"` (explicitly **not** `"unknown"`), and
  the module still returns full results from the local scanner. **The module
  works with zero API keys configured.**
- **Key configured** → numverify is queried concurrently with the local scanner.
  On success its `carrier` and `line_type` take precedence in the merged
  top-level verdict (it is the live, fraud-relevant signal the offline metadata
  often cannot supply — e.g. carrier is unavailable offline for
  number-portability regions like the US). On any network/HTTP/API failure it
  degrades to `status: "unknown"` with an error note while the local result is
  unaffected.
- **Swappable.** The endpoint is overridable via `NUMVERIFY_BASE_URL` (also used
  to point at a stub in tests), so a deployment can switch to numverify's HTTPS
  tier or a successor carrier-lookup provider without a code change — the
  isolation the decision doc asked for.

`numverify` errors are reported in-band (it returns `success:false` + an error
object, sometimes on an HTTP 200, e.g. for an invalid/exhausted key); those
degrade cleanly to `unknown`.

## I/O contract

- **Input (stdin):** a JSON object — a partial lead record with at least a
  `phone` field, expected in **international / E.164 form** (a leading `+` and
  country code; punctuation/spaces are tolerated and stripped). A bare national
  number with no country code cannot be resolved offline and degrades to
  `unknown`. All other fields are preserved untouched.
- **Output (stdout):** the same object with one key, `phone_validate`, added. The
  top-level fields are the merged verdict; the `local` and `numverify` blocks
  preserve each source's own output and status.

  | field | type | meaning |
  |---|---|---|
  | `status` | string | `"ok"` if the local scanner parsed the number; `"unknown"` otherwise |
  | `format_valid` | bool | number is a plausible, well-formed number (possible length/shape) |
  | `is_valid_number` | bool | number is valid/assignable per libphonenumber metadata |
  | `line_type` | string | `mobile` / `fixed_line` / `fixed_line_or_mobile` / `voip` / `toll_free` / `premium_rate` / `shared_cost` / `personal_number` / `pager` / `uan` / `voicemail` / `unknown` (numverify's value wins when present) |
  | `carrier` | string | carrier name (numverify's when present, else the offline lookup; `"unknown"` if neither resolves one) |
  | `country` | string | ISO 3166-1 alpha-2 region, e.g. `"US"`; `"unknown"` if unresolved |
  | `e164` | string | normalized E.164 form (present when parseable) |
  | `national` | string | national-format string (present when parseable) |
  | `country_code` | int | calling code, e.g. `1`, `44` |
  | `local` | object | the offline sub-result (`status`, the fields above, `checked_at`, `source_tool`, `error?`) |
  | `numverify` | object | the optional sub-result (`status: ok\|skipped\|unknown`, `valid?`, `line_type?`, `carrier?`, `country?`, `country_name?`, `location?`, `checked_at`, `source_tool`, `error?`) |
  | `checked_at` | string | RFC3339 UTC timestamp of the combined check |
  | `source_tools` | []string | sources that contributed (local always; numverify appended when not skipped) |

- **Audit (stderr):** exactly **two** JSON lines per call — **one per source**,
  always, even on failure — each carrying `tool` (name + version), `checked_at`,
  `phone` (**redacted**, e.g. `+14*******86`, so raw PII never lands in logs),
  `status`, and `legal_basis` (`GDPR Art.6(1)(f) legitimate-interest`, per
  [`docs/compliance.md`](../../docs/compliance.md), where phone OSINT is the
  "Low-Medium" personal-data-risk category). This satisfies the architecture
  "Audit" requirement for **both** sources: which tool/version ran, when, and the
  legal-basis tag.

### Failure mode

The module never crashes the pipeline. The two sources run **concurrently and
independently**: if numverify times out, is unconfigured, or errors, its
sub-result degrades to `skipped`/`unknown` while the local scanner still returns
— and a number the local scanner cannot parse yields `status: "unknown"` with an
`error` note rather than an exception. Each source is bounded by
`PHONE_VALIDATE_TIMEOUT` (a Go duration, default `10s`) applied **per source**,
and each is wrapped in a panic-recover. Exit code is `0` even on sub-source
failure; a **non-zero exit only** means stdin was not a readable JSON object.

## Run it

```bash
go build -o phone-validate ./cmd/phone-validate

echo '{"name":"Jane Doe","phone":"+14152007986","company":"Acme Corp"}' \
  | ./phone-validate
```

Real output (stdout), from a live run on 2026-07-13 with **no API key
configured** (numverify skipped, local scanner only):

```json
{
  "company": "Acme Corp",
  "name": "Jane Doe",
  "phone": "+14152007986",
  "phone_validate": {
    "status": "ok",
    "format_valid": true,
    "is_valid_number": true,
    "line_type": "fixed_line_or_mobile",
    "carrier": "unknown",
    "country": "US",
    "e164": "+14152007986",
    "national": "(415) 200-7986",
    "country_code": 1,
    "local": {
      "status": "ok",
      "format_valid": true,
      "is_valid_number": true,
      "line_type": "fixed_line_or_mobile",
      "country": "US",
      "e164": "+14152007986",
      "national": "(415) 200-7986",
      "country_code": 1,
      "checked_at": "2026-07-13T14:19:05Z",
      "source_tool": "nyaruka/phonenumbers@v2.0.3 (libphonenumber; PhoneInfoga local-scanner engine)"
    },
    "numverify": {
      "status": "skipped",
      "checked_at": "2026-07-13T14:19:05Z",
      "source_tool": "numverify (apilayer) /validate",
      "error": "NUMVERIFY_API_KEY not set — numverify carrier lookup skipped (local scanner still ran)"
    },
    "checked_at": "2026-07-13T14:19:05Z",
    "source_tools": [
      "nyaruka/phonenumbers@v2.0.3 (libphonenumber; PhoneInfoga local-scanner engine)"
    ]
  }
}
```

Audit lines on stderr for the same call (one per source, phone redacted):

```json
{"tool":"nyaruka/phonenumbers@v2.0.3 (libphonenumber; PhoneInfoga local-scanner engine)","checked_at":"2026-07-13T14:19:05Z","phone":"+14*******86","status":"ok","legal_basis":"GDPR Art.6(1)(f) legitimate-interest"}
{"tool":"numverify (apilayer) /validate","checked_at":"2026-07-13T14:19:05Z","phone":"+14*******86","status":"skipped","legal_basis":"GDPR Art.6(1)(f) legitimate-interest"}
```

### Offline carrier resolves for some regions

Same command with a UK mobile — the offline `carrier` subpackage resolves a
carrier name where number-portability rules allow it (captured live 2026-07-13,
still no API key):

```json
"phone_validate": {
  "status": "ok",
  "line_type": "mobile",
  "carrier": "Three",
  "country": "GB",
  "e164": "+447400123456",
  "national": "07400 123456",
  "country_code": 44,
  "...": "..."
}
```

### Clearly malformed number degrades gracefully

```bash
echo '{"phone":"not-a-phone"}' | ./phone-validate
```

```json
"phone_validate": {
  "status": "unknown",
  "format_valid": false,
  "is_valid_number": false,
  "line_type": "unknown",
  "carrier": "unknown",
  "country": "unknown",
  "local": {
    "status": "unknown",
    "error": "phone field contains no digits: \"not-a-phone\"",
    "...": "..."
  },
  "numverify": { "status": "skipped", "...": "..." }
}
```

### With numverify enabled

```bash
export NUMVERIFY_API_KEY=your_apilayer_key
echo '{"phone":"+14152007986"}' | ./phone-validate
```

The `numverify` block fills in and its live carrier/line-type win in the merged
verdict (real capture 2026-07-13 against a local numverify-schema stub — see
[Test](#test) — since this sandbox has no live key):

```json
"phone_validate": {
  "status": "ok",
  "line_type": "mobile",
  "carrier": "AT&T Mobility LLC",
  "country": "US",
  "numverify": {
    "status": "ok",
    "valid": true,
    "line_type": "mobile",
    "carrier": "AT&T Mobility LLC",
    "country": "US",
    "country_name": "United States of America",
    "location": "Novato",
    "source_tool": "numverify (apilayer) /validate"
  },
  "source_tools": [
    "nyaruka/phonenumbers@v2.0.3 (libphonenumber; PhoneInfoga local-scanner engine)",
    "numverify (apilayer) /validate"
  ]
}
```

## Configuration

| Env var | Default | Meaning |
|---|---|---|
| `PHONE_VALIDATE_TIMEOUT` | `10s` | Per-source timeout (Go duration, e.g. `5s`). |
| `NUMVERIFY_API_KEY` | *(unset)* | APILayer/numverify access key. Unset → numverify path skipped cleanly. |
| `NUMVERIFY_BASE_URL` | `https://apilayer.net/api/validate` | numverify endpoint override (swap provider / test stub). |

## Test

```bash
go test ./...
```

- `TestValidate_RealNumbers` runs the **real offline libphonenumber scanner** (no
  mocks) against a clearly valid E.164 number (`+14152007986`) and a clearly
  malformed one (`not-a-phone`), asserting on the actual output; numverify is
  skipped (no key). Fully offline — no network needed.
- `TestValidate_InvalidButParseable` covers the `format_valid && !is_valid_number`
  case (the 555 fictional US range).
- `TestValidate_MissingPhone` covers graceful degradation for an empty phone.
- `TestNumverify_StubServer` and `TestNumverify_APIErrorDegrades` exercise the
  **real numverify HTTP integration** against a local `httptest` stub (via
  `NUMVERIFY_BASE_URL`) returning canned numverify `/validate` JSON — proving the
  request/parse/merge path and the graceful degrade on an API error envelope,
  without needing a real API key.
- `TestRedact` covers PII redaction of the audit `phone` field.
- The CLI test (`cmd/phone-validate`) covers the full stdin→stdout contract,
  raw-field preservation, the two-audit-lines-on-stderr behavior (with redaction),
  and the only non-zero-exit path (unreadable stdin).

## Dependencies

- Go (built and tested with **1.24.0**; `go.mod` requires `go 1.24.0`, matching
  the `github.com/nyaruka/phonenumbers/v2` requirement).
- `github.com/nyaruka/phonenumbers/v2 v2.0.3` (**MIT**), pinned in `go.mod` —
  the maintained libphonenumber engine that powers the local scanner, including
  its `carrier` subpackage for offline carrier lookup.
- **No PhoneInfoga runtime dependency** (GPL-3.0, self-declared unmaintained —
  see [status](#phoneinfoga-maintenance-status--checked-live-2026-07-13)); its
  `local` scanner is used only as a design reference.
- **numverify — external, optional.** Only invoked when `NUMVERIFY_API_KEY` is
  set; unset → skipped. The module functions fully with **zero API keys, no
  accounts, no paid services**.
