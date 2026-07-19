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
// and degrades safely: missing binary, missing API key, timeout, parse error,
// or SSRF validation failure all produce a structured result with status
// "error"/"skipped" and never crash the pipeline.
package extraction

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
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
	BackendCrawl4AI  = "crawl4ai" // default
	BackendFirecrawl = "firecrawl"
)

// SourceTool identifiers, surfaced in results and audit records.
const (
	SourceToolCrawl4AI  = "unclecode/crawl4ai@v0.9.2 (CLI subprocess)"
	SourceToolFirecrawl = "firecrawl.dev/v1/scrape (HTTP adapter)"
)

// ToolVersion identifiers for the audit record.
const (
	ToolVersionCrawl4AI  = "crawl4ai==0.9.2"
	ToolVersionFirecrawl = "firecrawl.dev/v1"
)

// Input is the per-call extraction request. URL and PermissionRef are required;
// the other fields are passed through for context but are never required to crawl.
type Input struct {
	URL           string                 `json:"url"`
	PermissionRef string                 `json:"permission_ref"`
	SourceID      string                 `json:"source_id,omitempty"`
	Email         string                 `json:"email,omitempty"`
	Name          string                 `json:"name,omitempty"`
	Company       string                 `json:"company,omitempty"`
	Domain        string                 `json:"domain,omitempty"`
	Schema        map[string]interface{} `json:"schema,omitempty"`
}

// Fields holds the normalized, structured extraction output. Slice fields keep
// empty arrays (not omitted) so downstream consumers can rely on stable keys.
type Fields struct {
	CompanyName string   `json:"company_name,omitempty"`
	Emails      []string `json:"emails"`
	Phones      []string `json:"phones"`
	Addresses   []string `json:"addresses"`
	SocialLinks []string `json:"social_links"`
	ContactURLs []string `json:"contact_urls"`
	Description string   `json:"description,omitempty"`
	Title       string   `json:"title,omitempty"`
}

// Metadata carries non-PII runtime/config context.
type Metadata struct {
	Backend       string `json:"backend"`
	LegalBasis    string `json:"legal_basis"`
	PermissionRef string `json:"permission_ref"`
	HTTPStatus    int    `json:"http_status,omitempty"`
	Truncated     bool   `json:"truncated,omitempty"`
	Error         string `json:"error,omitempty"`
	RawBytes      int    `json:"raw_bytes,omitempty"`
	DurationMs    int64  `json:"duration_ms,omitempty"`
	LimitsApplied string `json:"limits_applied"`
}

// ProvenanceRecord links one extracted value back to the source URL and method.
type ProvenanceRecord struct {
	Field     string `json:"field"`
	Value     string `json:"value"`
	SourceURL string `json:"source_url"`
	Method    string `json:"method"`
	Timestamp string `json:"timestamp"`
}

// Result is the value stored under the lead's "extraction" key.
type Result struct {
	Status      string             `json:"status"` // "ok" | "partial" | "error" | "skipped"
	URL         string             `json:"url"`
	FinalURL    string             `json:"final_url,omitempty"`
	SourceTool  string             `json:"source_tool"`
	Confidence  float64            `json:"confidence"`
	Fields      Fields             `json:"fields"`
	RawMarkdown string             `json:"raw_markdown,omitempty"`
	Provenance  []ProvenanceRecord `json:"provenance,omitempty"`
	Metadata    Metadata           `json:"metadata"`
	Error       string             `json:"error,omitempty"`
	CheckedAt   string             `json:"checked_at"`
}

// AuditRecord is one structured audit-log line. The subject is the URL only —
// no raw email/name/company values, page content, or credentials are logged.
type AuditRecord struct {
	Module        string `json:"module"`
	Tool          string `json:"tool"`
	ToolVersion   string `json:"tool_version"`
	Timestamp     string `json:"timestamp"`
	LegalBasis    string `json:"legal_basis"`
	PermissionRef string `json:"permission_ref"`
	RequestURL    string `json:"request_url"`
	FinalURL      string `json:"final_url,omitempty"`
	Status        string `json:"status"`
	DurationMs    int64  `json:"duration_ms"`
	Limits        string `json:"limits"`
	Error         string `json:"error,omitempty"`
}

// runner is the shared backend-runner interface. It is implemented by the
// Crawl4AI subprocess runner and the Firecrawl HTTP adapter.
type runner interface {
	run(ctx context.Context, url string, timeout time.Duration) (Result, error)
	sourceTool() string
}

// Extractor runs a single-page extraction for one URL.
type Extractor struct {
	backend string
	runner  runner
	timeout time.Duration
	limiter *rateLimiter
	resolve func(string) ([]net.IP, error)
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
		resolve: net.LookupIP,
	}
}

// Extract performs the extraction for the given input. It returns the
// structured result, the audit record that should be logged, and no error:
// operational failures are reported in-band via Result.Status/Result.Error so
// the pipeline stays alive.
func (e *Extractor) Extract(ctx context.Context, in Input) (Result, AuditRecord) {
	start := time.Now()
	e.limiter.wait()
	now := time.Now().UTC()

	tool := e.runner.sourceTool()
	toolVersion := ToolVersionCrawl4AI
	if e.backend == BackendFirecrawl {
		toolVersion = ToolVersionFirecrawl
	}

	url := strings.TrimSpace(in.URL)
	permissionRef := strings.TrimSpace(in.PermissionRef)

	// Mandatory permission reference.
	if permissionRef == "" {
		res := Result{
			Status:     "skipped",
			URL:        url,
			SourceTool: tool,
			CheckedAt:  now.Format(time.RFC3339),
			Fields:     emptyFields(),
			Metadata:   Metadata{Backend: e.backend, LegalBasis: LegalBasis, PermissionRef: "", LimitsApplied: LimitsApplied},
			Error:      "no permission_ref provided",
		}
		audit := newAudit(tool, toolVersion, url, "", res.Status, res.Error, "", time.Since(start).Milliseconds(), now)
		e.limiter.backoff()
		return res, audit
	}

	// Empty URL after permission_ref is known.
	if url == "" {
		res := Result{
			Status:     "skipped",
			URL:        url,
			SourceTool: tool,
			CheckedAt:  now.Format(time.RFC3339),
			Fields:     emptyFields(),
			Metadata:   Metadata{Backend: e.backend, LegalBasis: LegalBasis, PermissionRef: permissionRef, LimitsApplied: LimitsApplied},
			Error:      "no url field present on lead record",
		}
		audit := newAudit(tool, toolVersion, url, "", res.Status, res.Error, permissionRef, time.Since(start).Milliseconds(), now)
		e.limiter.backoff()
		return res, audit
	}

	// SSRF / URL validation. We compute a result-sanitized URL (userinfo/fragment
	// stripped, query preserved) and an audit-sanitized URL (additionally
	// redacting query values) for every request.
	validatedURL, err := e.validateTargetURL(url)
	requestURLForResult := sanitizeURLStringResult(url)
	requestURLForAudit := sanitizeURLStringAudit(url)
	if validatedURL != nil {
		requestURLForResult = sanitizeURLForResult(validatedURL)
		requestURLForAudit = sanitizeURLForAudit(validatedURL)
	}

	if err != nil {
		res := Result{
			Status:     "skipped",
			URL:        requestURLForResult,
			SourceTool: tool,
			CheckedAt:  now.Format(time.RFC3339),
			Fields:     emptyFields(),
			Metadata:   Metadata{Backend: e.backend, LegalBasis: LegalBasis, PermissionRef: permissionRef, LimitsApplied: LimitsApplied},
			Error:      fmt.Sprintf("URL rejected by SSRF policy: %v", err),
		}
		audit := newAudit(tool, toolVersion, requestURLForAudit, "", res.Status, res.Error, permissionRef, time.Since(start).Milliseconds(), now)
		e.limiter.backoff()
		return res, audit
	}

	fetchURL := validatedURL.String()

	res, runErr := e.runner.run(ctx, fetchURL, e.timeout)
	if runErr != nil {
		if res.Status == "" || res.Status == "ok" {
			res.Status = "error"
		}
		if res.Error == "" {
			res.Error = runErr.Error()
		}
	}

	// Normalize result fields.
	normalizeFields(&res.Fields)
	res.URL = requestURLForResult
	finalURLForResult := requestURLForResult
	finalURLForAudit := requestURLForAudit
	if res.FinalURL != "" {
		finalURLForResult = sanitizeURLStringResult(res.FinalURL)
		finalURLForAudit = sanitizeURLStringAudit(res.FinalURL)
		res.FinalURL = finalURLForResult
	}
	if res.SourceTool == "" {
		res.SourceTool = tool
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
	res.Metadata.PermissionRef = permissionRef
	res.Metadata.LimitsApplied = LimitsApplied
	if res.Metadata.DurationMs == 0 {
		res.Metadata.DurationMs = time.Since(start).Milliseconds()
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

	// Build minimal provenance for extracted fields.
	if res.Status == "ok" || res.Status == "partial" {
		sourceURL := finalURLForResult
		if sourceURL == "" {
			sourceURL = requestURLForResult
		}
		res.Provenance = buildProvenance(res.Fields, sourceURL, e.backend, res.CheckedAt)
	}

	audit := newAudit(tool, toolVersion, requestURLForAudit, finalURLForAudit, res.Status, res.Error, permissionRef, time.Since(start).Milliseconds(), now)

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

// emptyFields returns a Fields value with all slice fields initialized to
// empty (non-nil) arrays. This keeps JSON output stable for downstream consumers.
func emptyFields() Fields {
	return Fields{
		Emails:      []string{},
		Phones:      []string{},
		Addresses:   []string{},
		SocialLinks: []string{},
		ContactURLs: []string{},
	}
}

// normalizeFields ensures that all slice fields are non-nil arrays. It is run
// on every result before it is returned so that JSON serialization emits []
// instead of null.
func normalizeFields(f *Fields) {
	if f.Emails == nil {
		f.Emails = []string{}
	}
	if f.Phones == nil {
		f.Phones = []string{}
	}
	if f.Addresses == nil {
		f.Addresses = []string{}
	}
	if f.SocialLinks == nil {
		f.SocialLinks = []string{}
	}
	if f.ContactURLs == nil {
		f.ContactURLs = []string{}
	}
}

func newAudit(tool, toolVersion, requestURL, finalURL, status, err, permissionRef string, durationMs int64, at time.Time) AuditRecord {
	return AuditRecord{
		Module:        "extraction",
		Tool:          tool,
		ToolVersion:   toolVersion,
		Timestamp:     at.Format(time.RFC3339),
		LegalBasis:    LegalBasis,
		PermissionRef: permissionRef,
		RequestURL:    requestURL,
		FinalURL:      finalURL,
		Status:        status,
		DurationMs:    durationMs,
		Limits:        LimitsApplied,
		Error:         err,
	}
}

// buildProvenance creates one record per extracted scalar value and one per
// item in slice fields, all linked to the source URL and method. It does not
// invent values: only observed, non-empty fields are recorded.
func buildProvenance(f Fields, sourceURL, method, timestamp string) []ProvenanceRecord {
	var out []ProvenanceRecord
	add := func(field, value string) {
		if value == "" {
			return
		}
		out = append(out, ProvenanceRecord{
			Field:     field,
			Value:     value,
			SourceURL: sourceURL,
			Method:    method,
			Timestamp: timestamp,
		})
	}

	add("company_name", f.CompanyName)
	add("title", f.Title)
	add("description", f.Description)
	for _, v := range f.Emails {
		add("emails", v)
	}
	for _, v := range f.Phones {
		add("phones", v)
	}
	for _, v := range f.Addresses {
		add("addresses", v)
	}
	for _, v := range f.SocialLinks {
		add("social_links", v)
	}
	for _, v := range f.ContactURLs {
		add("contact_urls", v)
	}

	return out
}

// ResultToJSON returns the result as indented JSON for tests / debugging.
func ResultToJSON(r Result) ([]byte, error) {
	return json.MarshalIndent(r, "", "  ")
}
