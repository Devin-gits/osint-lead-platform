// Package extraction implements the extraction module of the OSINT lead
// platform. Given a partial lead record with a URL, it fetches the landing page
// and extracts structured, low-risk fields (company name, contact links,
// emails, phones, description, title, etc.) while keeping the raw page content
// bounded. Two backends share one interface:
//
//   - crawl4ai (default): Apache-2.0 Python crawler invoked as a subprocess CLI
//     via wrapper/crawl4ai_extract.py. It is never imported into the MIT Go code.
//   - firecrawl (optional): thin Go HTTP adapter to the hosted Firecrawl API.
//     No Firecrawl source is vendored.
//
// Every call emits one structured audit line (URL, tool, status, legal basis)
// and degrades safely: missing binary, missing API key, timeout, or parse error
// all produce a structured result with status "error"/"skipped" and never crash
// the pipeline.
package extraction

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"time"
)

// LegalBasis is the documented GDPR basis for extracting low-risk business
// contact/identity signals from a lead's own landing page, per
// docs/compliance.md (Art. 6(1)(f) legitimate interest).
const LegalBasis = "GDPR Art.6(1)(f) legitimate-interest"

// DefaultTimeout bounds a single page crawl / API request (default 45s).
const DefaultTimeout = 45 * time.Second

// DefaultMinInterval is the minimum spacing between consecutive Extract calls
// on one Extractor (default 2s).
const DefaultMinInterval = 2 * time.Second

// MaxRawMarkdown caps the raw_markdown payload returned in the result to keep
// audit logs and downstream storage bounded.
const MaxRawMarkdown = 100 * 1024

// Backend constants — select via EXTRACTION_BACKEND env var.
const (
	BackendCrawl4AI = "crawl4ai" // default
	BackendFirecrawl = "firecrawl"
)

// SourceTool identifiers, surfaced in results and audit records.
const (
	SourceToolCrawl4AI  = "unclecode/crawl4ai@v0.9.2 (CLI subprocess)"
	SourceToolFirecrawl = "firecrawl.dev/v1/scrape (HTTP adapter)"
)

// Input is the per-call extraction request. Only URL is required; the other
// fields are passed through for context but are never required to crawl.
type Input struct {
	URL     string                 `json:"url"`
	Email   string                 `json:"email,omitempty"`
	Name    string                 `json:"name,omitempty"`
	Company string                 `json:"company,omitempty"`
	Domain  string                 `json:"domain,omitempty"`
	Schema  map[string]interface{} `json:"schema,omitempty"`
}

// Fields holds the normalized, structured extraction output.
type Fields struct {
	CompanyName  string   `json:"company_name,omitempty"`
	Emails       []string `json:"emails,omitempty"`
	Phones       []string `json:"phones,omitempty"`
	Addresses    []string `json:"addresses,omitempty"`
	SocialLinks  []string `json:"social_links,omitempty"`
	ContactURLs  []string `json:"contact_urls,omitempty"`
	Description  string   `json:"description,omitempty"`
	Title        string   `json:"title,omitempty"`
}

// Metadata carries non-PII runtime/config context.
type Metadata struct {
	Backend       string `json:"backend"`
	LegalBasis    string `json:"legal_basis"`
	HTTPStatus    int    `json:"http_status,omitempty"`
	Truncated     bool   `json:"truncated,omitempty"`
	Error         string `json:"error,omitempty"`
	RawBytes      int    `json:"raw_bytes,omitempty"`
}

// Result is the value stored under the lead's "extraction" key.
type Result struct {
	Status      string   `json:"status"` // "ok" | "partial" | "error" | "skipped"
	URL         string   `json:"url"`
	FinalURL    string   `json:"final_url,omitempty"`
	SourceTool  string   `json:"source_tool"`
	Confidence  float64  `json:"confidence"`
	Fields      Fields   `json:"fields"`
	RawMarkdown string   `json:"raw_markdown,omitempty"`
	Metadata    Metadata `json:"metadata"`
	Error       string   `json:"error,omitempty"`
	CheckedAt   string   `json:"checked_at"`
}

// AuditRecord is one structured audit-log line. The subject is the URL only —
// no raw email/name/company values are logged.
type AuditRecord struct {
	Tool       string `json:"tool"`
	URL        string `json:"url"`
	CheckedAt  string `json:"checked_at"`
	Status     string `json:"status"`
	LegalBasis string `json:"legal_basis"`
	Error      string `json:"error,omitempty"`
}

// runner is the shared backend-runner interface. It is implemented by the
// Crawl4AI subprocess runner and the Firecrawl HTTP adapter.
type runner interface {
	run(ctx context.Context, url string, timeout time.Duration) (Result, error)
	sourceTool() string
}

// Extractor runs a single-page extraction for one URL.
type Extractor struct {
	backend   string
	runner    runner
	timeout   time.Duration
	limiter   *rateLimiter
}

// NewExtractor builds an Extractor with the default (Crawl4AI) backend. The
// EXTRACTION_BACKEND env var takes precedence over the backend parameter.
func NewExtractor(timeout, minInterval time.Duration, backend string) *Extractor {
	if env := os.Getenv("EXTRACTION_BACKEND"); env != "" {
		backend = env
	}
	if backend == "" {
		backend = BackendCrawl4AI
	}
	if timeout <= 0 {
		timeout = DefaultTimeout
	}
	if minInterval <= 0 {
		minInterval = DefaultMinInterval
	}

	var r runner
	switch backend {
	case BackendFirecrawl:
		r = newFirecrawlRunner()
	default:
		backend = BackendCrawl4AI
		r = newCrawl4AIRunner()
	}

	return &Extractor{
		backend: backend,
		runner:  r,
		timeout: timeout,
		limiter: newRateLimiter(minInterval),
	}
}

// Extract performs the extraction for the given input. It returns the
// structured result, the audit record that should be logged, and no error:
// operational failures are reported in-band via Result.Status/Result.Error so
// the pipeline stays alive.
func (e *Extractor) Extract(ctx context.Context, in Input) (Result, AuditRecord) {
	e.limiter.wait()
	now := time.Now().UTC()

	url := strings.TrimSpace(in.URL)
	if url == "" {
		res := Result{
			Status:    "skipped",
			URL:       url,
			SourceTool: e.runner.sourceTool(),
			CheckedAt: now.Format(time.RFC3339),
			Metadata:  Metadata{Backend: e.backend, LegalBasis: LegalBasis},
			Error:     "no url field present on lead record",
		}
		audit := newAudit(e.runner.sourceTool(), url, res.Status, res.Error, now)
		e.limiter.backoff()
		return res, audit
	}

	res, err := e.runner.run(ctx, url, e.timeout)
	if err != nil {
		if res.Status == "" || res.Status == "ok" {
			res.Status = "error"
		}
		if res.Error == "" {
			res.Error = err.Error()
		}
	}

	res.URL = url
	if res.SourceTool == "" {
		res.SourceTool = e.runner.sourceTool()
	}
	if res.CheckedAt == "" {
		res.CheckedAt = now.Format(time.RFC3339)
	}
	if res.Metadata.Backend == "" {
		res.Metadata.Backend = e.backend
	}
	if res.Metadata.LegalBasis == "" {
		res.Metadata.LegalBasis = LegalBasis
	}
	if res.Confidence == 0 && res.Status == "ok" {
		res.Confidence = confidenceFor(res.Fields)
	}

	// Bound raw markdown payload.
	if len(res.RawMarkdown) > MaxRawMarkdown {
		res.RawMarkdown = res.RawMarkdown[:MaxRawMarkdown]
		res.Metadata.Truncated = true
	}
	res.Metadata.RawBytes = len(res.RawMarkdown)

	audit := newAudit(e.runner.sourceTool(), url, res.Status, res.Error, now)

	if res.Status == "ok" || res.Status == "partial" {
		e.limiter.reset()
	} else {
		e.limiter.backoff()
	}

	return res, audit
}

// confidenceFor returns a simple 0.0–1.0 score based on how many field
// categories have at least one non-empty value. It is intentionally
// conservative: a single structured signal is useful, but more signals increase
// confidence.
func confidenceFor(f Fields) float64 {
	score := 0
	if f.CompanyName != "" {
		score++
	}
	if f.Title != "" {
		score++
	}
	if f.Description != "" {
		score++
	}
	if len(f.Emails) > 0 {
		score++
	}
	if len(f.Phones) > 0 {
		score++
	}
	if len(f.SocialLinks) > 0 {
		score++
	}
	if len(f.ContactURLs) > 0 {
		score++
	}
	max := 7.0
	if score == 0 {
		return 0.0
	}
	return float64(score) / max
}

func newAudit(tool, url, status, err string, at time.Time) AuditRecord {
	return AuditRecord{
		Tool:       tool,
		URL:        url,
		CheckedAt:  at.Format(time.RFC3339),
		Status:     status,
		LegalBasis: LegalBasis,
		Error:      err,
	}
}

// ResultToJSON returns the result as indented JSON for tests / debugging.
func ResultToJSON(r Result) ([]byte, error) {
	return json.MarshalIndent(r, "", "  ")
}
