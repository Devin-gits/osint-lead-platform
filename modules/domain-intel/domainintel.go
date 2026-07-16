// Package domainintel implements the domain-intel module of the OSINT lead
// platform. Given a partial lead record with a "domain" field, it produces the
// Ingest-stage domain profile defined in docs/architecture.md, adding a
// namespaced "domain_intel" key that combines two independent sub-tools:
//
//   - web_check: a lightweight reimplementation of lissy93/web-check's DNS/TLS/
//     HTTP/WHOIS checks ("is this an established, real business domain?"), using
//     only the Go standard library.
//   - harvester: laramies/theHarvester invoked as an external CLI subprocess
//     ("what hosts/subdomains/contacts hang off this domain?"), never imported
//     as a library, restricted to a keyless non-breach-database source
//     allowlist.
//
// Per the Stage 1 decision (docs/decisions/stage-1-decision.md, domain-intel
// section) both tools run. Each sub-tool degrades to status "unknown"
// independently on failure/timeout without blocking the other or crashing the
// pipeline. Every call emits one audit record per tool.
package domainintel

import (
	"context"
	"sync"
	"time"
)

// LegalBasis is the documented GDPR basis for domain intelligence on a lead's
// own (consented) domain, per docs/compliance.md (Art. 6(1)(f) legitimate
// interest; domain/DNS/WHOIS intel is the "Low" personal-data-risk category).
// Logged on every call for both tools to satisfy the architecture "Audit"
// requirement.
const LegalBasis = "GDPR Art.6(1)(f) legitimate-interest"

// DefaultTimeout bounds each sub-tool independently (not the pair). web-check's
// DNS/TLS/HTTP/WHOIS network calls are usually sub-second, but theHarvester
// fans out across several third-party sources and empirically takes several
// seconds; on expiry that tool degrades to "unknown" while the other still returns.
const DefaultTimeout = 60 * time.Second

// Result is the value stored under the lead's "domain_intel" key. It namespaces
// each tool's sub-result and records when the combined check ran and which
// tools contributed, matching the sub-schema in the README.
type Result struct {
	WebCheck    WebCheckResult  `json:"web_check"`
	Harvester   HarvesterResult `json:"harvester"`
	CheckedAt   string          `json:"checked_at"`
	SourceTools []string        `json:"source_tools"`
}

// AuditRecord is one structured audit-log line. One is emitted per underlying
// tool per call, regardless of outcome: which tool/version ran, when, the
// domain, the resulting status/error, and the legal basis — the facts
// docs/architecture.md and docs/compliance.md require be logged per run.
type AuditRecord struct {
	Tool       string `json:"tool"`
	CheckedAt  string `json:"checked_at"`
	Domain     string `json:"domain"`
	Status     string `json:"status"`
	LegalBasis string `json:"legal_basis"`
	Error      string `json:"error,omitempty"`
}

// Analyzer runs the domain-intel checks. Construct with NewAnalyzer. It is safe
// to reuse across calls.
type Analyzer struct {
	timeout time.Duration
}

// NewAnalyzer builds an Analyzer. Pass a non-positive timeout to use
// DefaultTimeout. The timeout applies to each sub-tool independently.
func NewAnalyzer(timeout time.Duration) *Analyzer {
	if timeout <= 0 {
		timeout = DefaultTimeout
	}
	return &Analyzer{timeout: timeout}
}

// Analyze runs both sub-tools against domain concurrently and returns the
// combined Result plus one AuditRecord per tool. It never returns an error:
// each sub-tool reports failures in-band via its own Status "unknown" and Error
// fields, so a failure in either tool keeps the pipeline alive and does not
// block the other tool's result.
func (a *Analyzer) Analyze(domain string) (Result, []AuditRecord) {
	now := time.Now().UTC()
	ctx := context.Background()

	var (
		wg  sync.WaitGroup
		web WebCheckResult
		har HarvesterResult
	)
	wg.Add(2)
	go func() {
		defer wg.Done()
		web = safeWebCheck(ctx, domain, a.timeout)
	}()
	go func() {
		defer wg.Done()
		har = safeHarvester(ctx, domain, a.timeout)
	}()
	wg.Wait()

	res := Result{
		WebCheck:    web,
		Harvester:   har,
		CheckedAt:   now.Format(time.RFC3339),
		SourceTools: []string{WebCheckTool, HarvesterTool},
	}
	audits := []AuditRecord{
		newAudit(WebCheckTool, domain, web.Status, web.Error, now),
		newAudit(HarvesterTool, domain, har.Status, har.Error, now),
	}
	return res, audits
}

// safeWebCheck runs the web-check sub-tool with a panic recover so an
// unexpected panic degrades that sub-result to "unknown" instead of taking down
// the whole call.
func safeWebCheck(ctx context.Context, domain string, timeout time.Duration) (res WebCheckResult) {
	defer func() {
		if r := recover(); r != nil {
			res = WebCheckResult{
				Status:     "unknown",
				CheckedAt:  time.Now().UTC().Format(time.RFC3339),
				SourceTool: WebCheckTool,
				Error:      recoverMsg(r),
			}
		}
	}()
	return runWebCheck(ctx, domain, timeout)
}

func safeHarvester(ctx context.Context, domain string, timeout time.Duration) (res HarvesterResult) {
	defer func() {
		if r := recover(); r != nil {
			res = HarvesterResult{
				Status:     "unknown",
				Hosts:      []Host{},
				IPs:        []string{},
				Emails:     []string{},
				Sources:    filteredHarvesterSources(),
				CheckedAt:  time.Now().UTC().Format(time.RFC3339),
				SourceTool: HarvesterTool,
				Error:      recoverMsg(r),
			}
		}
	}()
	return runHarvester(ctx, domain, timeout)
}

func recoverMsg(r interface{}) string {
	return "recovered from panic in sub-tool: " + toString(r)
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

func newAudit(tool, domain, status, errorNote string, at time.Time) AuditRecord {
	return AuditRecord{
		Tool:       tool,
		CheckedAt:  at.Format(time.RFC3339),
		Domain:     normalizeDomain(domain),
		Status:     status,
		LegalBasis: LegalBasis,
		Error:      errorNote,
	}
}
