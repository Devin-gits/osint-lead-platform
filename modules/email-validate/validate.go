// Package emailvalidate implements the email-validate module of the OSINT lead
// platform. It wraps AfterShip's email-verifier Go library behind the module
// contract defined in docs/architecture.md: it takes a partial lead record,
// runs syntax / MX / disposable-free-role checks (no email is sent, SMTP probe
// off by default), and returns the same record with a namespaced
// "email_validate" key added. It never panics the pipeline: any failure yields
// status "unknown" with an error note.
package emailvalidate

import (
	"context"
	"fmt"
	"strings"
	"time"

	emailverifier "github.com/AfterShip/email-verifier"
)

// SourceTool identifies the underlying engine and pinned version. Emitted in
// every result and in every audit-log line so runs are attributable.
const SourceTool = "AfterShip/email-verifier@v1.4.1"

// LegalBasis is the documented GDPR basis for processing lead emails, per
// docs/compliance.md (Art. 6(1)(f) legitimate interest / anti-fraud). Logged on
// every call to satisfy the architecture "Audit" requirement.
const LegalBasis = "GDPR Art.6(1)(f) legitimate-interest"

// DefaultTimeout bounds a single verification. The non-SMTP path is single-digit
// milliseconds, but DNS/MX lookups can stall; on expiry we degrade to "unknown"
// rather than block the pipeline.
const DefaultTimeout = 10 * time.Second

// Result is the value stored under the lead's "email_validate" key. Field shape
// is adapted from email-verifier's real Result struct (see verifier.go): the
// booleans mirror Disposable/RoleAccount/Free, deliverable mirrors the
// Reachable enum, and syntax_valid / has_mx_records carry the deliverability
// signals. Deliverable is a tri-state string ("yes"/"no"/"unknown") because
// with the SMTP probe disabled the library cannot assert true mailbox
// reachability — reporting a bare bool would overstate confidence.
type Result struct {
	Status         string `json:"status"`      // "ok" once the check ran; "unknown" on any failure
	Deliverable    string `json:"deliverable"` // mirrors email-verifier Reachable: "yes"|"no"|"unknown"
	SyntaxValid    bool   `json:"syntax_valid"`
	HasMXRecords   bool   `json:"has_mx_records"`
	IsDisposable   bool   `json:"is_disposable"`
	IsRoleAccount  bool   `json:"is_role_account"`
	IsFreeProvider bool   `json:"is_free_provider"`
	CheckedAt      string `json:"checked_at"`  // RFC3339 UTC
	SourceTool     string `json:"source_tool"` // == SourceTool
	Error          string `json:"error,omitempty"`
}

// AuditRecord is one structured audit-log line, emitted for every call
// regardless of outcome. It records which tool/version ran, when, the lead
// email checked, the final status, and the legal basis — the four things
// docs/architecture.md and docs/compliance.md require be logged per run.
type AuditRecord struct {
	Tool       string `json:"tool"`
	CheckedAt  string `json:"checked_at"`
	Email      string `json:"email"`
	Status     string `json:"status"`
	LegalBasis string `json:"legal_basis"`
}

// Validator wraps a configured email-verifier instance. Construct with
// NewValidator. It is safe to reuse across calls.
type Validator struct {
	verifier *emailverifier.Verifier
	timeout  time.Duration
}

// NewValidator builds a Validator with the SMTP deliverability probe disabled
// (the compliance-safe default from the evaluation) and domain-typo suggestion
// enabled. Pass a non-positive timeout to use DefaultTimeout.
func NewValidator(timeout time.Duration) *Validator {
	if timeout <= 0 {
		timeout = DefaultTimeout
	}
	v := emailverifier.NewVerifier().
		DisableSMTPCheck().
		EnableDomainSuggest()
	return &Validator{verifier: v, timeout: timeout}
}

// Validate runs the verifier against email and returns a Result plus the
// AuditRecord that should be logged for this call. It never returns an error:
// timeouts, empty input, and library failures are all reported in-band via
// Result.Status == "unknown" and Result.Error, keeping the pipeline alive.
func (val *Validator) Validate(email string) (Result, AuditRecord) {
	now := time.Now().UTC()
	res := Result{
		Status:      "unknown",
		Deliverable: "unknown",
		CheckedAt:   now.Format(time.RFC3339),
		SourceTool:  SourceTool,
	}

	email = strings.TrimSpace(email)
	if email == "" {
		res.Error = "no email field present on lead record"
		return res, val.audit(email, res.Status, now)
	}

	ret, err := val.verifyWithTimeout(email)
	if err != nil {
		res.Error = err.Error()
		return res, val.audit(email, res.Status, now)
	}

	res.Status = "ok"
	res.Deliverable = ret.Reachable
	res.SyntaxValid = ret.Syntax.Valid
	res.HasMXRecords = ret.HasMxRecords
	res.IsDisposable = ret.Disposable
	res.IsRoleAccount = ret.RoleAccount
	res.IsFreeProvider = ret.Free
	return res, val.audit(email, res.Status, now)
}

// verifyWithTimeout runs the (synchronous) library call in a goroutine and
// abandons it if val.timeout elapses, so a stalled DNS/MX lookup cannot hang
// the caller. A panic inside the library is recovered and surfaced as an error.
func (val *Validator) verifyWithTimeout(email string) (*emailverifier.Result, error) {
	ctx, cancel := context.WithTimeout(context.Background(), val.timeout)
	defer cancel()

	type outcome struct {
		ret *emailverifier.Result
		err error
	}
	ch := make(chan outcome, 1)

	go func() {
		defer func() {
			if r := recover(); r != nil {
				ch <- outcome{err: fmt.Errorf("verifier panicked: %v", r)}
			}
		}()
		ret, err := val.verifier.Verify(email)
		ch <- outcome{ret: ret, err: err}
	}()

	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("verification timed out after %s", val.timeout)
	case o := <-ch:
		if o.err != nil {
			return nil, fmt.Errorf("verifier error: %w", o.err)
		}
		return o.ret, nil
	}
}

func (val *Validator) audit(email, status string, at time.Time) AuditRecord {
	return AuditRecord{
		Tool:       SourceTool,
		CheckedAt:  at.Format(time.RFC3339),
		Email:      email,
		Status:     status,
		LegalBasis: LegalBasis,
	}
}
