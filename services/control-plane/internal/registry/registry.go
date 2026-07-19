// Package registry holds module metadata and availability information for the
// control-plane API. It does not execute modules; that is the runner's job.
package registry

import (
	"strconv"
	"strings"

	"github.com/Moyeil-73/osint-lead-platform/modules/social-footprint"
	"github.com/Moyeil-73/osint-lead-platform/services/control-plane/internal/models"
)

// Registry is a static catalogue of modules. Wired status is represented by the
// runner, not here, so the catalogue stays honest about what the API advertises.
type Registry struct {
	modules []models.ModuleInfo
	docs    map[string]string
}

// New returns the production registry.
func New() *Registry {
	modules := []models.ModuleInfo{
		{
			Name:          models.ModuleEmailValidate,
			DisplayName:   "Email Validation",
			Category:      "validate",
			DevStatus:     "available",
			NamespacedKey: "email_validate",
			BackingTools:  []string{"AfterShip/email-verifier@v1.4.1"},
			Description:   "Syntax, MX, disposable and role-account checks. SMTP probe disabled for compliance.",
			MinInputField: "email",
			RiskLevelNote: "Low: validates the email address without sending mail.",
		},
		{
			Name:          models.ModulePhoneValidate,
			DisplayName:   "Phone Validation",
			Category:      "validate",
			DevStatus:     "available",
			NamespacedKey: "phone_validate",
			BackingTools:  []string{"nyaruka/phonenumbers@v1.5.0", "numverify (optional)"},
			Description:   "Offline libphonenumber parsing with optional numverify carrier lookup.",
			MinInputField: "phone",
			RiskLevelNote: "Low-Medium: line type, carrier and validity. Phone numbers are redacted in audit logs.",
		},
		{
			Name:          models.ModuleDomainIntel,
			DisplayName:   "Domain Intelligence",
			Category:      "ingest",
			DevStatus:     "available",
			NamespacedKey: "domain_intel",
			BackingTools:  []string{"lissy93/web-check (reimplemented)", "laramies/theHarvester (subprocess)"},
			Description:   "DNS, TLS, WHOIS and public-subdomain signals for a lead's domain.",
			MinInputField: "domain",
			RiskLevelNote: "Low: domain/DNS/WHOIS data is public business context.",
		},
		{
			Name:          models.ModuleSocialFootprint,
			DisplayName:   "Social Footprint",
			Category:      "validate",
			DevStatus:     "available",
			NamespacedKey: "social_footprint",
			BackingTools:  []string{"soxoj/maigret@0.6.2 (Python wrapper subprocess, curated platform list)"},
			Description:   "Per-handle match/no-match spot check across a curated platform allow-list.",
			MinInputField: "email",
			RiskLevelNote: "Medium: only public handle presence is captured, never scraped profile fields.",
		},
		{
			Name:          models.ModuleExtraction,
			DisplayName:   "Lead Extraction",
			Category:      "ingest",
			DevStatus:     "available",
			NamespacedKey: "extraction",
			BackingTools:  []string{"unclecode/crawl4ai@v0.9.2 (CLI subprocess)", "firecrawl.dev/v1/scrape (optional HTTP adapter)"},
			Description:   "Extract low-risk public fields (company, emails, phones, social/contact links, title, description) from one customer-supplied, permissioned public URL.",
			MinInputField: "url",
			RiskLevelNote: "Low-Medium: fetches a public page; SSRF controls, no auth crawl, no recursion, raw content bounded to 100 KB.",
		},
		{
			Name:          models.ModuleCompanyEnrich,
			DisplayName:   "Company Enrichment",
			Category:      "enrich",
			DevStatus:     "available",
			NamespacedKey: "company_enrich",
			BackingTools:  []string{"company-enrich/local (deterministic)", "discolike (optional paid adapter)"},
			Description:   "Enrich company firmographics from deterministic public sources and optional paid B2B data adapters. Requires at least domain, company, or url. Optional DISCOLIKE_API_KEY for paid adapter.",
			MinInputField: "domain",
			RiskLevelNote: "Low-Medium: company-level data only; paid adapters require API key and are optional.",
		},
	}

	docs := map[string]string{
		models.ModuleEmailValidate: "Runs AfterShip/email-verifier with the SMTP probe disabled. " +
			"Results are namespaced under the `email_validate` key. Audit records include the email checked, tool version, status and legal basis.",
		models.ModulePhoneValidate: "Runs the offline libphonenumber-based local scanner. " +
			"If NUMVERIFY_API_KEY is set an optional carrier lookup is attempted. Phone numbers are redacted in audit logs.",
		models.ModuleDomainIntel: "Runs a Go reimplementation of lissy93/web-check's DNS/TLS/HTTP/WHOIS checks and, " +
			"if theHarvester is installed on PATH (or DOMAIN_INTEL_HARVESTER_BIN is set), invokes it as a subprocess with a keyless source allowlist. " +
			"Each sub-tool degrades to status 'unknown' independently on failure or timeout; the combined 'domain_intel' status is 'ok' if either sub-tool reports ok. " +
			"Results are namespaced under the `domain_intel` key. Audit records include the domain, tool, status, legal basis and any error.",
		models.ModuleSocialFootprint: "Derives up to " + strconv.Itoa(socialfootprint.MaxHandles) + " handle candidates from the email local-part and any already-enriched domain_intel.harvester sub-object, " +
			"then runs the Maigret Python wrapper as a subprocess over a curated platform allow-list. " +
			"If Python or the wrapper is unavailable, the module degrades each handle to status 'unknown' with an error note but does not crash the pipeline. " +
			"Each handle produces one audit record with the handle checked, tool, status and legal basis. " +
			"Results are namespaced under the `social_footprint` key.",
		models.ModuleExtraction: "Fetches one permissioned public URL per lead and extracts low-risk business contact/identity signals. " +
			"Requires `url` and `permission_ref`. Default backend is Crawl4AI (local Python subprocess); Firecrawl is optional via `EXTRACTION_BACKEND=firecrawl` and `FIRECRAWL_API_KEY`. " +
			"SSRF controls block private/link-local/metadata IPs, non-standard ports, and credentialed URLs. Raw markdown is capped at 100 KB and not written to audit logs.",
		models.ModuleCompanyEnrich: "Enriches company firmographics. Requires at least `domain`, `company`, or `url` plus `permission_ref`. " +
		"The local provider is deterministic and no-key; the optional DiscoLike adapter uses `DISCOLIKE_API_KEY`. " +
		"Results are namespaced under `company_enrich`. Audit records include the domain, tool, status, legal basis, and limits.",
	}

	return &Registry{modules: modules, docs: docs}
}

// List returns all modules.
func (r *Registry) List() []models.ModuleInfo {
	return r.modules
}

// Get returns a module detail, or false if the name is unknown.
func (r *Registry) Get(name string) (models.ModuleDetail, bool) {
	for _, m := range r.modules {
		if strings.EqualFold(m.Name, name) {
			d := models.ModuleDetail{ModuleInfo: m}
			if doc, ok := r.docs[m.Name]; ok {
				d.Docs = doc
			}
			return d, true
		}
	}
	return models.ModuleDetail{}, false
}

// IsKnown reports whether name is a recognised module name.
func (r *Registry) IsKnown(name string) bool {
	_, ok := r.Get(name)
	return ok
}
