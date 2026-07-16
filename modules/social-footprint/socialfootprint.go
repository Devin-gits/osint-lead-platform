// Package socialfootprint implements the social-footprint module of the OSINT
// lead platform. Given a partial (or partially-enriched) lead record with at
// least an "email" field, it produces the Validate-stage "is this a real,
// active person?" signal defined in docs/architecture.md, adding a namespaced
// "social_footprint" key with per-handle, per-platform claimed/unclaimed
// signals from Maigret and/or Sherlock.
//
// Integration shape. Maigret is a Python tool; the prior modules are Go. Per the
// Stage 1 decision (docs/decisions/stage-1-decision.md -> social-footprint) and
// evaluations/maigret.md, Maigret is embedded as a Python LIBRARY (its MIT
// license permits this) inside a small wrapper script (wrapper/maigret_check.py)
// which this Go module invokes as a SUBPROCESS. The subprocess boundary is only
// the Go<->Python bridge and mirrors the pattern domain-intel already uses for
// theHarvester; it keeps the repo's module-runner uniform (single static Go
// binary, stdin->stdout) while the actual Maigret calls are library calls, not
// CLI shell-outs.
//
// The handle dependency. Maigret validates a username, but the raw lead schema
// (name, email, phone, company, domain) has no username. This module derives
// candidate handles internally as a first-class step (see handles.go): primarily
// from the email local-part (plus 2-3 cheap variants), and optionally from an
// already-present domain_intel.harvester sub-object when the input is an enriched
// record. If no usable handle can be derived, the module degrades to
// status "skipped" (correctly nothing to check) rather than "unknown".
//
// Compliance guardrails (enforced in code, not just docs), per the decision doc
// and docs/compliance.md's "Social footprint = Medium" row:
//   - Backend selector: SOCIAL_FOOTPRINT_BACKEND chooses Maigret (default),
//     Sherlock, or consensus merge. Each backend uses its own curated allow-list.
//   - Scope cap: only a curated allow-list of major platforms is checked,
//     never Maigret's 3000+ site default or Sherlock's 400+ site default
//     (see curatedPlatforms, sherlockCuratedPlatforms, and the wrappers).
//   - Handle cap: at most MaxHandles candidates are checked per lead, bounding
//     fan-out.
//   - Rate limit: an in-process limiter (see ratelimit.go) enforces a minimum
//     delay between consecutive per-lead invocations when Check is called in a loop.
//   - Minimal collection: only match/no-match + URL is captured, never scraped
//     profile fields (bio/location/linked accounts) — the wrappers disable
//     parsing and recursion.
package socialfootprint

import (
	"context"
	"math"
	"os"
	"strconv"
	"time"
)

// LegalBasis is the documented GDPR basis for a social-footprint spot check on a
// consented lead, per docs/compliance.md (Art. 6(1)(f) legitimate interest;
// social footprint is the "Medium" personal-data-risk category — confirming a
// lead's online presence is real, as an anti-fraud signal). Logged on every call
// to satisfy the architecture "Audit" requirement.
const LegalBasis = "GDPR Art.6(1)(f) legitimate-interest"

// SourceTool identifies the underlying engine and the version this module's
// output parsing was verified against.
const SourceTool = "soxoj/maigret@0.6.2 (embedded Python library via wrapper subprocess)"

// DefaultTimeout bounds a single handle's Maigret subprocess run. The default is
// generous because one run fans out (bounded) HTTP requests to the curated
// platform list; on expiry that handle degrades to "unknown" without blocking
// other handles or crashing the pipeline.
const DefaultTimeout = 90 * time.Second

// Env var names referenced inside the package.
const (
	timeoutEnv     = "SOCIAL_FOOTPRINT_TIMEOUT"
	minIntervalEnv = "SOCIAL_FOOTPRINT_MIN_INTERVAL"
)

// DefaultMinInterval is the minimum spacing enforced between consecutive per-lead
// Check calls on the same Validator (the in-process rate limiter). It makes the
// "rate-limited, per-lead spot check — never a bulk sweep" requirement a
// code-level guardrail. The first call on a fresh Validator is never delayed.
const DefaultMinInterval = 5 * time.Second

// MaxHandles caps how many derived handle candidates are actually checked per
// lead, bounding the fan-out of a single call (each handle is one Maigret run
// over the curated platform list).
const MaxHandles = 3

// Backend constants — select via SOCIAL_FOOTPRINT_BACKEND env var.
const (
	BackendMaigret   = "maigret" // default
	BackendSherlock  = "sherlock"
	BackendBoth      = "both"      // consensus merge
	BackendOsintgram = "osintgram" // optional Instagram depth; GPL-3.0 CLI subprocess only

	// SourceToolSherlock identifies the Sherlock engine in audit records.
	SourceToolSherlock = "sherlock-project/sherlock@0.16.1 (embedded Python library via wrapper subprocess)"

	// SourceToolOsintgram identifies the Osintgram engine in audit records.
	// Osintgram is GPL-3.0 and is invoked as a subprocess CLI only, never as an
	// imported library, so the MIT platform code remains license-clean.
	SourceToolOsintgram = "Datalux/Osintgram (GPL-3.0 CLI subprocess; command=info only)"
)

// DefaultOsintgramMinInterval is the recommended minimum spacing for the
// Osintgram backend. Instagram is more aggressive about anti-automation than
// Maigret's multi-site spot check, so the default is higher unless the operator
// explicitly overrides it via SOCIAL_FOOTPRINT_MIN_INTERVAL.
const DefaultOsintgramMinInterval = 15 * time.Second

// statusOK / statusSkipped / statusUnknown are the module-level status values.
// "skipped" means correctly nothing to check (no derivable handle); "unknown"
// means a check was attempted but failed — the distinction the task requires.
const (
	statusOK      = "ok"
	statusSkipped = "skipped"
	statusUnknown = "unknown"
)

// InstagramDetails is the minimal, compliance-reviewed enrichment the Osintgram
// backend may attach to an Instagram PlatformSignal. It contains only counts and
// boolean flags; it deliberately omits biography text, full contact strings, HD
// images, GPS, and follower/following graphs.
type InstagramDetails struct {
	UserID         string `json:"user_id,omitempty"`
	IsPrivate      bool   `json:"is_private"`
	IsVerified     bool   `json:"is_verified"`
	IsBusiness     bool   `json:"is_business"`
	FollowerCount  int    `json:"follower_count,omitempty"`
	FollowingCount int    `json:"following_count,omitempty"`
	MediaCount     int    `json:"media_count,omitempty"`
	HasPublicEmail bool   `json:"has_public_email"`
	CheckedVia     string `json:"checked_via"`
}

// PlatformSignal is one platform's result for one handle: whether the handle is
// claimed/available/unknown on that platform, and (for a claimed hit) the public
// profile URL. Deliberately no scraped profile fields — see package doc. The
// optional InstagramDetails field is used only by the Osintgram backend and is
// omitted by Maigret/Sherlock.
type PlatformSignal struct {
	Platform   string            `json:"platform"`
	Status     string            `json:"status"` // "claimed" | "available" | "unknown"
	URL        string            `json:"url,omitempty"`
	HTTPStatus int               `json:"http_status,omitempty"`
	Instagram  *InstagramDetails `json:"instagram,omitempty"`
}

// HandleResult is the per-candidate-handle block: which platforms were checked
// and their signals. Status is "ok" if the Maigret run completed (even with some
// per-platform "unknown"s) and "unknown" if the run itself failed.
type HandleResult struct {
	Handle       string           `json:"handle"`
	Origin       string           `json:"origin"` // how the handle was derived (see handles.go)
	Status       string           `json:"status"` // "ok" | "unknown"
	Platforms    []PlatformSignal `json:"platforms"`
	ClaimedCount int              `json:"claimed_count"`
	CheckedAt    string           `json:"checked_at"`
	SourceTool   string           `json:"source_tool"`
	Error        string           `json:"error,omitempty"`
}

// SocialFootprintResult is the value stored under the lead's "social_footprint" key.
type SocialFootprintResult struct {
	Status         string         `json:"status"`           // "ok" | "skipped"
	Reason         string         `json:"reason,omitempty"` // set when Status == "skipped"
	HandlesChecked []string       `json:"handles_checked"`  // the handle strings actually checked
	Handles        []HandleResult `json:"handles,omitempty"`
	ActiveSignals  int            `json:"active_signals"` // total "claimed" hits across all handles
	Confidence     float64        `json:"confidence"`     // 0.0-1.0 ratio of claimed hits to the active backend's primary platform count
	Metadata       map[string]any `json:"metadata"`       // config/runtime facts for reviewers
	CheckedAt      string         `json:"checked_at"`
	SourceTool     string         `json:"source_tool"`
	RateLimitNote  string         `json:"rate_limit_note"`
}

// AuditRecord is one structured audit-log line. One is emitted per handle
// actually checked (or a single record noting the skip), carrying the
// tool/version, timestamp, the handle checked (the only PII surfaced — never the
// raw email/name), the resulting status, and the legal basis.
type AuditRecord struct {
	Tool       string `json:"tool"`
	CheckedAt  string `json:"checked_at"`
	Handle     string `json:"handle"`
	Status     string `json:"status"`
	LegalBasis string `json:"legal_basis"`
}

// Validator runs social-footprint checks. Construct with NewValidator. It is safe
// to reuse across leads; reuse is in fact how the in-process rate limiter spaces
// out consecutive per-lead calls.
type Validator struct {
	timeout   time.Duration
	limiter   *rateLimiter
	runner    maigretRunner // pluggable so tests can inject a fake instead of Python
	backend   string        // active backend: BackendMaigret, BackendSherlock, or BackendBoth
	platforms []string      // platform allow-list passed to runner.run() for this backend
}

// NewValidator builds a Validator with the default (Maigret) backend.
// SOCIAL_FOOTPRINT_BACKEND env var can override the backend at runtime.
func NewValidator(timeout, minInterval time.Duration) *Validator {
	return NewValidatorWithBackend(timeout, minInterval, "")
}

// NewValidatorWithBackend builds a Validator with an explicit backend choice.
// SOCIAL_FOOTPRINT_BACKEND env var takes precedence over the backend parameter.
// An empty backend string defaults to BackendMaigret.
func NewValidatorWithBackend(timeout, minInterval time.Duration, backend string) *Validator {
	if env := os.Getenv("SOCIAL_FOOTPRINT_BACKEND"); env != "" {
		backend = env
	}
	if backend == "" {
		backend = BackendMaigret
	}
	if timeout <= 0 {
		timeout = DefaultTimeout
	}
	if minInterval <= 0 {
		// Instagram is more aggressive than the multi-site presence checkers, so
		// Osintgram gets a higher default min interval unless the operator explicitly
		// overrides it via SOCIAL_FOOTPRINT_MIN_INTERVAL.
		if backend == BackendOsintgram && os.Getenv(minIntervalEnv) == "" {
			minInterval = DefaultOsintgramMinInterval
		} else {
			minInterval = DefaultMinInterval
		}
	}

	var runner maigretRunner
	var platforms []string
	switch backend {
	case BackendSherlock:
		runner = &sherlockRunner{}
		platforms = sherlockCuratedPlatforms
	case BackendBoth:
		runner = &bothRunner{
			primary:   &subprocessRunner{},
			secondary: &sherlockRunner{},
		}
		platforms = curatedPlatforms
	case BackendOsintgram:
		runner = &osintgramRunner{}
		platforms = []string{"Instagram"}
	default: // BackendMaigret and unknown values
		backend = BackendMaigret
		runner = &subprocessRunner{}
		platforms = curatedPlatforms
	}

	return &Validator{
		timeout:   timeout,
		limiter:   newRateLimiter(minInterval),
		runner:    runner,
		backend:   backend,
		platforms: platforms,
	}
}

// sourceTool returns the engine identifier surfaced in results, handle blocks,
// and audit records. It agrees with the active backend so every visible surface
// reports which engine actually ran.
func (v *Validator) sourceTool() string {
	switch v.backend {
	case BackendSherlock:
		return SourceToolSherlock
	case BackendBoth:
		return SourceTool + " + " + SourceToolSherlock + " consensus"
	case BackendOsintgram:
		return SourceToolOsintgram
	default:
		return SourceTool
	}
}

// rateLimitNote is the compliance-relevant note embedded in every result,
// documenting the scope/rate discipline this module enforces.
func (v *Validator) rateLimitNote() string {
	scopeNote := "curated " + strconv.Itoa(len(v.platforms)) + "-platform allow-list"
	if v.backend == BackendBoth {
		scopeNote = "Maigret " + scopeNote + " + Sherlock " + strconv.Itoa(len(sherlockCuratedPlatforms)) + "-platform allow-list"
	}
	if v.backend == BackendOsintgram {
		scopeNote = "Instagram single-platform allow-list (Osintgram command=info only)"
	}
	return "per-lead rate-limited spot check (min " + v.limiter.interval().String() +
		" between leads, exponential backoff on consecutive errors); scope hard-capped to a " + scopeNote +
		" and " + strconv.Itoa(MaxHandles) + " handle candidates; recursion, profile scraping, and " +
		"proxy/Cloudflare block-evasion disabled; " + LegalBasis
}

// confidence computes a simple 0.0-1.0 ratio of claimed signals to the
// theoretical maximum for this call. It is intentionally conservative: even one
// hit on a curated public platform is a meaningful signal, but a saturated
// profile (claimed on many platforms) is a stronger one.
//
// The denominator uses v.platforms so it matches the active backend's primary
// coverage. In BackendBoth mode we deliberately keep Maigret's curated list as
// the denominator — a conservative choice that avoids double-counting the
// overlapping Sherlock coverage.
func (v *Validator) confidence(handleCount, claimed int) float64 {
	if handleCount == 0 {
		return 0
	}
	max := handleCount * len(v.platforms)
	if max == 0 {
		return 0
	}
	return math.Round(float64(claimed)/float64(max)*1000) / 1000
}

// metadata returns runtime/config facts for reviewers: scope, rate-limit base,
// and legal basis. It is a stable snapshot per call and never includes PII.
func (v *Validator) metadata() map[string]any {
	m := map[string]any{
		"source_tool":        v.sourceTool(),
		"legal_basis":        LegalBasis,
		"max_handles":        MaxHandles,
		"platform_count":     len(v.platforms),
		"rate_limit_base":    v.limiter.minInterval.String(),
		"rate_limit_current": v.limiter.interval().String(),
	}
	if v.backend == BackendBoth {
		m["sherlock_platform_count"] = len(sherlockCuratedPlatforms)
	}
	return m
}

// Check derives candidate handles from the lead, runs a rate-limited, scope-
// capped spot check for each, and returns the combined SocialFootprintResult
// plus one AuditRecord per handle checked (or a single record for a skip). It never
// returns an error: every failure is reported in-band so the pipeline stays
// alive.
func (v *Validator) Check(lead map[string]interface{}) (SocialFootprintResult, []AuditRecord) {
	now := time.Now().UTC()

	candidates := deriveHandles(lead)
	if len(candidates) > MaxHandles {
		candidates = candidates[:MaxHandles]
	}

	if len(candidates) == 0 {
		res := SocialFootprintResult{
			Status:         statusSkipped,
			Reason:         "no usable handle could be derived from the lead (needs an email local-part or an enriched domain_intel.harvester sub-object)",
			HandlesChecked: []string{},
			Confidence:     0,
			Metadata:       v.metadata(),
			CheckedAt:      now.Format(time.RFC3339),
			SourceTool:     v.sourceTool(),
			RateLimitNote:  v.rateLimitNote(),
		}
		audit := AuditRecord{
			Tool:       v.sourceTool(),
			CheckedAt:  now.Format(time.RFC3339),
			Handle:     "",
			Status:     statusSkipped,
			LegalBasis: LegalBasis,
		}
		return res, []AuditRecord{audit}
	}

	// Rate-limit at the per-lead boundary: one wait covers the whole lead's
	// (bounded) set of handle checks, so leads are spaced, not individual
	// platform requests.
	v.limiter.wait()

	res := SocialFootprintResult{
		Status:         statusOK,
		HandlesChecked: make([]string, 0, len(candidates)),
		Handles:        make([]HandleResult, 0, len(candidates)),
		Metadata:       v.metadata(),
		CheckedAt:      now.Format(time.RFC3339),
		SourceTool:     v.sourceTool(),
		RateLimitNote:  v.rateLimitNote(),
	}
	audits := make([]AuditRecord, 0, len(candidates))

	hadError := false
	for _, c := range candidates {
		hr := v.checkHandle(c, now)
		res.Handles = append(res.Handles, hr)
		res.HandlesChecked = append(res.HandlesChecked, c.handle)
		res.ActiveSignals += hr.ClaimedCount
		if hr.Status == statusUnknown {
			hadError = true
		}
		audits = append(audits, AuditRecord{
			Tool:       v.sourceTool(),
			CheckedAt:  now.Format(time.RFC3339),
			Handle:     c.handle,
			Status:     hr.Status,
			LegalBasis: LegalBasis,
		})
	}

	res.Confidence = v.confidence(len(res.HandlesChecked), res.ActiveSignals)
	if hadError {
		v.limiter.backoff()
	} else {
		v.limiter.reset()
	}
	return res, audits
}

// checkHandle runs one backend spot check for a single handle over the curated
// platform allow-list, wrapped with a per-handle timeout and a panic-recover so a
// single handle failure degrades to "unknown" instead of taking down the call.
func (v *Validator) checkHandle(c handleCandidate, now time.Time) (hr HandleResult) {
	hr = HandleResult{
		Handle:     c.handle,
		Origin:     c.origin,
		Status:     statusUnknown,
		Platforms:  []PlatformSignal{},
		CheckedAt:  now.Format(time.RFC3339),
		SourceTool: v.sourceTool(),
	}
	defer func() {
		if r := recover(); r != nil {
			hr.Status = statusUnknown
			hr.Error = "recovered from panic in runner: " + toString(r)
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), v.timeout)
	defer cancel()

	out, err := v.runner.run(ctx, c.handle, v.platforms, v.timeout)
	if err != nil {
		hr.Error = err.Error()
		v.limiter.backoff()
		return hr
	}
	if out.Error != "" {
		hr.Error = v.backend + " wrapper: " + out.Error
		// Fall through: partial results (if any) are still surfaced below.
	}
	for _, r := range out.Results {
		sig := PlatformSignal{
			Platform:   r.Platform,
			Status:     r.Status,
			URL:        r.URL,
			HTTPStatus: r.HTTPStatus,
		}
		hr.Platforms = append(hr.Platforms, sig)
		if r.Status == "claimed" {
			hr.ClaimedCount++
		}
	}
	if out.Error == "" {
		hr.Status = statusOK
	}
	return hr
}

func toString(v interface{}) string {
	switch t := v.(type) {
	case string:
		return t
	case error:
		return t.Error()
	default:
		return "unknown panic"
	}
}
