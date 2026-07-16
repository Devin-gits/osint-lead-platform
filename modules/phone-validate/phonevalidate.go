// Package phonevalidate implements the phone-validate module of the OSINT lead
// platform. Given a partial lead record with a "phone" field, it produces the
// Validate-stage phone profile defined in docs/architecture.md, adding a
// namespaced "phone_validate" key that combines two independent signal sources:
//
//   - local: an offline scanner (number validity, format, country, line type,
//     and — where derivable offline — carrier) built directly on
//     github.com/nyaruka/phonenumbers, the maintained MIT Go port of Google's
//     libphonenumber. This is the same engine PhoneInfoga's `local` scanner
//     wraps; see the README for why we adopt the engine directly rather than
//     importing PhoneInfoga (GPL-3.0) into this MIT module.
//   - numverify: an OPTIONAL third-party carrier-lookup API, invoked only when
//     NUMVERIFY_API_KEY is set. Unset → the sub-result reports "skipped" (not
//     "unknown") and the module still returns full results from the local
//     scanner. Per the Stage 1 decision this is a thin, swappable dependency.
//
// Per the Stage 1 decision (docs/decisions/stage-1-decision.md, phone-validate
// section) the local scanner is the primary integration and numverify is a
// watched, optional add-on. Each source degrades to status "unknown"
// independently on failure/timeout without blocking the other or crashing the
// pipeline. Every call emits one audit record per source.
package phonevalidate

import (
	"context"
	"sync"
	"time"
)

// LegalBasis is the documented GDPR basis for phone validation on a consented
// lead, per docs/compliance.md (Art. 6(1)(f) legitimate interest; phone OSINT
// is the "Low-Medium" personal-data-risk category — carrier/line-type lookups
// are standard fraud-prevention practice). Logged on every call for both
// sources to satisfy the architecture "Audit" requirement.
const LegalBasis = "GDPR Art.6(1)(f) legitimate-interest"

// DefaultTimeout bounds each source independently (not the pair). The local
// scanner is pure offline parsing and effectively instantaneous; numverify does
// a network round-trip and on expiry degrades to "unknown" while the local
// result still returns.
const DefaultTimeout = 10 * time.Second

// unknown is the sentinel used for any field the sources could not resolve,
// matching the architecture contract's "mark the field as unknown" requirement.
const unknown = "unknown"

// Result is the value stored under the lead's "phone_validate" key. The
// top-level answer fields are the module's merged verdict; the per-source blocks
// (Local, Numverify) preserve each source's own output and status for
// transparency and auditing. Field selection follows what the actual tools
// return (libphonenumber + numverify), not invented fields.
type Result struct {
	Status        string          `json:"status"`          // "ok" if the local scanner parsed the number; else "unknown"
	FormatValid   bool            `json:"format_valid"`    // number is a plausible, well-formed number (possible length/shape)
	IsValidNumber bool            `json:"is_valid_number"` // number is valid per libphonenumber metadata (assignable)
	LineType      string          `json:"line_type"`       // merged: numverify if present, else local; "unknown" if neither
	Carrier       string          `json:"carrier"`         // merged: numverify if present, else local offline; "unknown" if none
	Country       string          `json:"country"`         // ISO 3166-1 alpha-2 region, e.g. "US"; "unknown" if unresolved
	E164          string          `json:"e164,omitempty"`  // normalized E.164 form, when parseable
	National      string          `json:"national,omitempty"`
	CountryCode   int32           `json:"country_code,omitempty"` // calling code, e.g. 1, 44
	RiskFlags     []string        `json:"risk_flags,omitempty"`   // fraud-relevant signals derived from line_type/validity/carrier
	Local         LocalResult     `json:"local"`
	Numverify     NumverifyResult `json:"numverify"`
	CheckedAt     string          `json:"checked_at"`   // RFC3339 UTC
	SourceTools   []string        `json:"source_tools"` // sources that contributed (local always; numverify when not skipped)
}

// AuditRecord is one structured audit-log line. One is emitted per underlying
// source per call, regardless of outcome: which tool/version ran, when, the
// (redacted) number, the resulting status, and the legal basis — the facts
// docs/architecture.md and docs/compliance.md require be logged per run.
type AuditRecord struct {
	Tool       string `json:"tool"`
	CheckedAt  string `json:"checked_at"`
	Phone      string `json:"phone"` // redacted (see redact) so raw PII does not leak into logs
	Status     string `json:"status"`
	LegalBasis string `json:"legal_basis"`
}

// Validator runs the phone-validate checks. Construct with NewValidator. It is
// safe to reuse across calls.
type Validator struct {
	timeout   time.Duration
	numverify *numverifyClient
}

// NewValidator builds a Validator. Pass a non-positive timeout to use
// DefaultTimeout (applied to each source independently). The numverify client is
// configured from the environment (NUMVERIFY_API_KEY / NUMVERIFY_BASE_URL); with
// no key set, the numverify path is skipped cleanly.
func NewValidator(timeout time.Duration) *Validator {
	if timeout <= 0 {
		timeout = DefaultTimeout
	}
	return &Validator{timeout: timeout, numverify: newNumverifyClientFromEnv()}
}

// Validate runs both sources against phone concurrently and returns the combined
// Result plus one AuditRecord per source. It never returns an error: each source
// reports failures in-band via its own Status "unknown" (or "skipped" for
// numverify), so a failure in either keeps the pipeline alive and does not block
// the other source's result.
func (v *Validator) Validate(phone string) (Result, []AuditRecord) {
	now := time.Now().UTC()
	ctx := context.Background()

	var (
		wg  sync.WaitGroup
		loc LocalResult
		nv  NumverifyResult
	)
	wg.Add(2)
	go func() {
		defer wg.Done()
		loc = safeLocal(phone, now)
	}()
	go func() {
		defer wg.Done()
		nv = safeNumverify(ctx, v.numverify, phone, v.timeout, now)
	}()
	wg.Wait()

	res := merge(loc, nv, now)
	audits := []AuditRecord{
		newAudit(LocalTool, phone, loc.Status, now),
		newAudit(NumverifyTool, phone, nv.Status, now),
	}
	return res, audits
}

// merge builds the top-level verdict from the two sub-results. The local scanner
// is authoritative for format/validity/country/normalization (offline, always
// available, deterministic). For line_type and carrier, numverify's live lookup
// takes precedence when it succeeded, since it is the fraud-relevant signal the
// local offline metadata often cannot supply (e.g. carrier is unavailable
// offline for number-portability regions like the US); otherwise the local
// value is used, falling back to "unknown".
func merge(loc LocalResult, nv NumverifyResult, now time.Time) Result {
	res := Result{
		Status:        loc.Status,
		FormatValid:   loc.FormatValid,
		IsValidNumber: loc.IsValid,
		LineType:      firstNonEmpty(nv.lineTypeIfOK(), loc.LineType, unknown),
		Carrier:       firstNonEmpty(nv.carrierIfOK(), loc.Carrier, unknown),
		Country:       firstNonEmpty(loc.Country, nv.countryIfOK(), unknown),
		E164:          loc.E164,
		National:      loc.National,
		CountryCode:   loc.CountryCode,
		Local:         loc,
		Numverify:     nv,
		CheckedAt:     now.Format(time.RFC3339),
		SourceTools:   []string{LocalTool},
	}
	if nv.Status != StatusSkipped {
		res.SourceTools = append(res.SourceTools, NumverifyTool)
	}
	res.RiskFlags = deriveRiskFlags(res)
	return res
}

// deriveRiskFlags extracts simple, fraud-relevant risk signals from the merged
// result. It is intentionally conservative: it flags line types commonly abused
// for throw-away/fraud accounts and numbers where neither source could resolve a
// carrier or confirm validity. The list is empty for ordinary, well-formed
// mobile/fixed-line numbers with a known carrier.
func deriveRiskFlags(r Result) []string {
	var flags []string
	if !r.IsValidNumber {
		flags = append(flags, "invalid_number")
	}
	switch r.LineType {
	case "voip":
		flags = append(flags, "voip")
	case "toll_free":
		flags = append(flags, "toll_free")
	case "premium_rate":
		flags = append(flags, "premium_rate")
	case "pager":
		flags = append(flags, "pager")
	}
	if r.Numverify.Status == StatusOK && (r.Carrier == "" || r.Carrier == unknown) {
		flags = append(flags, "carrier_unknown")
	}
	return flags
}

// safeLocal runs the local scanner with a panic recover so an unexpected panic
// degrades that sub-result to "unknown" instead of taking down the whole call.
func safeLocal(phone string, now time.Time) (res LocalResult) {
	defer func() {
		if r := recover(); r != nil {
			res = LocalResult{
				Status:     unknown,
				LineType:   unknown,
				Country:    unknown,
				CheckedAt:  now.Format(time.RFC3339),
				SourceTool: LocalTool,
				Error:      "recovered from panic in local scanner: " + toString(r),
			}
		}
	}()
	return runLocal(phone, now)
}

// safeNumverify runs the numverify client with a panic recover, degrading to
// "unknown" on panic. When no API key is configured the client is nil and the
// result is "skipped".
func safeNumverify(ctx context.Context, c *numverifyClient, phone string, timeout time.Duration, now time.Time) (res NumverifyResult) {
	defer func() {
		if r := recover(); r != nil {
			res = NumverifyResult{
				Status:     unknown,
				CheckedAt:  now.Format(time.RFC3339),
				SourceTool: NumverifyTool,
				Error:      "recovered from panic in numverify client: " + toString(r),
			}
		}
	}()
	return c.run(ctx, phone, timeout, now)
}

func newAudit(tool, phone, status string, at time.Time) AuditRecord {
	return AuditRecord{
		Tool:       tool,
		CheckedAt:  at.Format(time.RFC3339),
		Phone:      redact(phone),
		Status:     status,
		LegalBasis: LegalBasis,
	}
}

func firstNonEmpty(vals ...string) string {
	for _, s := range vals {
		if s != "" {
			return s
		}
	}
	return ""
}

func toString(v interface{}) string {
	if s, ok := v.(string); ok {
		return s
	}
	if e, ok := v.(error); ok {
		return e.Error()
	}
	return "unknown panic"
}
