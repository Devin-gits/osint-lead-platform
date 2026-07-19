// Package companyenrich implements the company enrichment module of the OSINT
// lead platform. It takes a partial lead record and enriches it with public,
// company-level firmographics. It never scrapes LinkedIn, never uses LLMs to
// synthesize facts, and never fetches people/CEO/founder/job data.
//
// The default local provider is deterministic and requires no API key.
// Optional paid adapters (DiscoLike) are invoked only when configured and only
// to fill gaps after the local provider.
package companyenrich

import (
	"context"
	"encoding/json"
	"strings"
	"time"
)

// LegalBasis is the documented GDPR basis for company-level enrichment.
const LegalBasis = "GDPR Art.6(1)(f) legitimate-interest"

// DefaultTimeout bounds a single enrichment run (default 30s).
const DefaultTimeout = 30 * time.Second

// DefaultMinInterval enforces a small spacing between consecutive Enrich calls
// on a reused Enricher (default 500ms).
const DefaultMinInterval = 500 * time.Millisecond

// LimitsApplied is a human-readable summary of the guardrails enforced.
const LimitsApplied = "timeout=30s,min_interval=500ms,providers=local,discolike"

// Input is the per-call enrichment request.
type Input struct {
	Domain         string             `json:"domain,omitempty"`
	Company        string             `json:"company,omitempty"`
	URL            string             `json:"url,omitempty"`
	PermissionRef  string             `json:"permission_ref"`
	SourceID       string             `json:"source_id,omitempty"`
	Extraction     *ExtractionInput   `json:"extraction,omitempty"`
	DomainIntel    *DomainIntelInput  `json:"domain_intel,omitempty"`
	RequiredFields []string           `json:"required_fields,omitempty"`
}

// ExtractionInput is the subset of the extraction result the local provider uses.
type ExtractionInput struct {
	Status string           `json:"status"`
	Fields ExtractionFields `json:"fields"`
}

// ExtractionFields is the subset of extraction.fields we consume.
type ExtractionFields struct {
	CompanyName string   `json:"company_name,omitempty"`
	Title       string   `json:"title,omitempty"`
	Description string   `json:"description,omitempty"`
	SocialLinks []string `json:"social_links"`
}

// DomainIntelInput is the subset of the domain-intel result the local provider uses.
type DomainIntelInput struct {
	Status    string              `json:"status"`
	WebCheck  DomainIntelWebCheck `json:"web_check"`
	CheckedAt string              `json:"checked_at"`
}

// DomainIntelWebCheck mirrors the web_check sub-result.
type DomainIntelWebCheck struct {
	Status string             `json:"status"`
	SSL    *DomainIntelSSL   `json:"ssl,omitempty"`
	Whois  *DomainIntelWhois `json:"whois,omitempty"`
}

// DomainIntelSSL captures the certificate subject.
type DomainIntelSSL struct {
	Subject string `json:"subject,omitempty"`
}

// DomainIntelWhois captures registrar and creation signals.
type DomainIntelWhois struct {
	Registrar     string `json:"registrar,omitempty"`
	CreatedDate   string `json:"created_date,omitempty"`
	DomainAgeDays int    `json:"domain_age_days"`
}

// Headquarters carries company location fields.
type Headquarters struct {
	City    string `json:"city,omitempty"`
	State   string `json:"state,omitempty"`
	Country string `json:"country,omitempty"`
	Address string `json:"address,omitempty"`
}

// Fields is the enriched company output.
type Fields struct {
	Domain             string            `json:"domain,omitempty"`
	Name               string            `json:"name,omitempty"`
	LegalName          string            `json:"legal_name,omitempty"`
	Website            string            `json:"website,omitempty"`
	Description        string            `json:"description,omitempty"`
	Founded            *int              `json:"founded,omitempty"`
	EmployeeCount      *int              `json:"employee_count,omitempty"`
	EmployeeCountRange string            `json:"employee_count_range,omitempty"`
	Headquarters       *Headquarters     `json:"headquarters,omitempty"`
	Industry           []string          `json:"industry,omitempty"`
	SocialLinks        map[string]string `json:"social_links,omitempty"`
	TechStack          []string          `json:"tech_stack,omitempty"`
	Sources            []string          `json:"sources,omitempty"`
}

// Metadata carries non-PII runtime/config context.
type Metadata struct {
	Backend       string  `json:"backend"`
	LegalBasis    string  `json:"legal_basis"`
	PermissionRef string  `json:"permission_ref"`
	DurationMs    int64   `json:"duration_ms,omitempty"`
	LimitsApplied string  `json:"limits_applied"`
	Error         string  `json:"error,omitempty"`
}

// Result is the value stored under the lead's "company_enrich" key.
type Result struct {
	Status     string   `json:"status"`
	SourceTool string   `json:"source_tool"`
	Confidence float64  `json:"confidence"`
	Fields     Fields   `json:"fields"`
	Metadata   Metadata `json:"metadata"`
	Error      string   `json:"error,omitempty"`
	CheckedAt  string   `json:"checked_at"`
}

// AuditRecord is one structured audit-log line.
type AuditRecord struct {
	Module        string    `json:"module"`
	Tool          string    `json:"tool"`
	ToolVersion   string    `json:"tool_version"`
	Timestamp     string    `json:"timestamp"`
	LegalBasis    string    `json:"legal_basis"`
	PermissionRef string    `json:"permission_ref"`
	Subject       Subject   `json:"subject"`
	Status        string    `json:"status"`
	DurationMs    int64     `json:"duration_ms"`
	Limits        string    `json:"limits"`
	Error         string    `json:"error,omitempty"`
}

// Subject identifies the target of an audit line; no personal data.
type Subject struct {
	Domain  string `json:"domain,omitempty"`
	Company string `json:"company,omitempty"`
}

// Enricher runs providers in order and merges results.
type Enricher struct {
	timeout     time.Duration
	minInterval time.Duration
	providers   []Provider
	clock       func() time.Time
}

// NewEnricher builds an Enricher with the default provider set.
func NewEnricher(timeout, minInterval time.Duration) *Enricher {
	if timeout <= 0 {
		timeout = DefaultTimeout
	}
	if minInterval <= 0 {
		minInterval = DefaultMinInterval
	}
	return &Enricher{
		timeout:     timeout,
		minInterval: minInterval,
		providers: []Provider{
			newLocalProvider(),
			newDiscolikeProvider(),
		},
		clock: time.Now,
	}
}

// SetProviders replaces the provider list (useful for tests).
func (e *Enricher) SetProviders(providers ...Provider) {
	e.providers = providers
}

// Enrich validates input, runs providers, merges, and returns the result plus
// audit records. It never returns an error for operational outcomes; all
// failures are reported in-band via Result.Status/Error.
func (e *Enricher) Enrich(ctx context.Context, in Input) (Result, []AuditRecord) {
	start := e.clock()
	now := start.UTC()

	permissionRef := strings.TrimSpace(in.PermissionRef)
	if permissionRef == "" {
		res := Result{
			Status:     "skipped",
			SourceTool: "company-enrich",
			Confidence: 0,
			Fields:     emptyFields(),
			Metadata:   emptyMetadata(permissionRef),
			Error:      "missing permission_ref",
			CheckedAt:  now.Format(time.RFC3339),
		}
		audit := newAudit("company-enrich", "company-enrich", permissionRef, Subject{}, "skipped", "missing permission_ref", 0, now)
		return res, []AuditRecord{audit}
	}

	domain, company, url := normalizeInputs(in.Domain, in.Company, in.URL)
	if domain == "" && company == "" && url == "" {
		res := Result{
			Status:     "skipped",
			SourceTool: "company-enrich",
			Confidence: 0,
			Fields:     emptyFields(),
			Metadata:   emptyMetadata(permissionRef),
			Error:      "missing domain, company, and url",
			CheckedAt:  now.Format(time.RFC3339),
		}
		audit := newAudit("company-enrich", "company-enrich", permissionRef, Subject{}, "skipped", "missing domain, company, and url", 0, now)
		return res, []AuditRecord{audit}
	}

	normalizedIn := in
	normalizedIn.Domain = domain
	normalizedIn.Company = company
	normalizedIn.URL = url
	normalizedIn.PermissionRef = permissionRef

	subject := Subject{Domain: domain, Company: company}
	if subject.Domain == "" && subject.Company == "" {
		subject.Domain = url
	}

	required := defaultP0()
	if len(normalizedIn.RequiredFields) > 0 {
		required = normalizedIn.RequiredFields
	}

	ctx, cancel := context.WithTimeout(ctx, e.timeout)
	defer cancel()

	merged, providerAudits, _, errMsg := runWaterfall(ctx, e.providers, normalizedIn, required, e.clock, subject, permissionRef)

	res := Result{
		SourceTool: sourceToolFor(merged.Sources),
		Fields:     merged,
		Metadata:   emptyMetadata(permissionRef),
		CheckedAt:  e.clock().UTC().Format(time.RFC3339),
	}
	res.Metadata.DurationMs = e.clock().Sub(start).Milliseconds()

	switch {
	case fieldsSatisfied(merged, defaultP0()):
		res.Status = "ok"
	case hasUsefulData(merged):
		res.Status = "partial"
	case errMsg != "":
		res.Status = "error"
		res.Error = errMsg
	default:
		res.Status = "partial"
		res.Error = "no useful enrichment data returned"
	}
	if errMsg != "" && res.Status == "ok" {
		res.Error = errMsg
	}

	res.Confidence = confidenceFor(merged)

	audits := providerAudits
	audits = append(audits, newAudit(
		"company-enrich",
		"company-enrich",
		permissionRef,
		subject,
		res.Status,
		res.Error,
		res.Metadata.DurationMs,
		now,
	))

	return res, audits
}

func normalizeInputs(domain, company, url string) (string, string, string) {
	domain = strings.TrimSpace(domain)
	company = strings.TrimSpace(company)
	url = strings.TrimSpace(url)

	if url != "" && !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		url = "https://" + url
	}
	if url != "" && domain == "" {
		domain = domainFromURL(url)
	}
	if domain != "" && url == "" {
		url = "https://" + domain
	}
	return domain, company, url
}

func domainFromURL(raw string) string {
	s := strings.TrimPrefix(raw, "https://")
	s = strings.TrimPrefix(s, "http://")
	s = strings.TrimPrefix(s, "www.")
	idx := strings.IndexAny(s, "/:?#")
	if idx >= 0 {
		s = s[:idx]
	}
	return strings.ToLower(strings.TrimSpace(s))
}

func humanizeDomain(domain string) string {
	parts := strings.Split(domain, ".")
	if len(parts) == 0 {
		return domain
	}
	base := parts[0]
	base = strings.Title(base)
	return base
}

func emptyFields() Fields {
	return Fields{
		Industry:    []string{},
		TechStack:   []string{},
		Sources:     []string{},
		SocialLinks: map[string]string{},
	}
}

func emptyMetadata(permissionRef string) Metadata {
	return Metadata{
		Backend:       "company-enrich",
		LegalBasis:    LegalBasis,
		PermissionRef: permissionRef,
		LimitsApplied: LimitsApplied,
	}
}

func newAudit(module, tool, permissionRef string, subject Subject, status, err string, durationMs int64, at time.Time) AuditRecord {
	toolVersion := "company-enrich/local"
	if tool == "discolike" {
		toolVersion = "discolike/v1"
	}
	return AuditRecord{
		Module:        module,
		Tool:          tool,
		ToolVersion:   toolVersion,
		Timestamp:     at.Format(time.RFC3339),
		LegalBasis:    LegalBasis,
		PermissionRef: permissionRef,
		Subject:       subject,
		Status:        status,
		DurationMs:    durationMs,
		Limits:        LimitsApplied,
		Error:         err,
	}
}

func defaultP0() []string {
	return []string{"domain", "name", "website"}
}

// ResultToJSON returns the result as indented JSON for tests / debugging.
func ResultToJSON(r Result) ([]byte, error) {
	return json.MarshalIndent(r, "", "  ")
}

// asIntPointer returns a pointer to v for JSON nullable ints.
func asIntPointer(v int) *int { return &v }
