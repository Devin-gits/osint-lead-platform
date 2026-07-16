// Package registry holds module metadata and availability information for the
// control-plane API. It does not execute modules; that is the runner's job.
package registry

import (
	"strings"

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
			DevStatus:     "in_development",
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
			DevStatus:     "in_development",
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
			DevStatus:     "planned",
			NamespacedKey: "extraction",
			BackingTools:  []string{},
			Description:   "Parse and normalise raw lead payloads into the canonical schema.",
			MinInputField: "raw_payload",
			RiskLevelNote: "Low: normalisation only.",
		},
		{
			Name:          models.ModuleCompanyEnrich,
			DisplayName:   "Company Enrichment",
			Category:      "enrich",
			DevStatus:     "planned",
			NamespacedKey: "company_enrich",
			BackingTools:  []string{},
			Description:   "Enrich company context from public business registries and datasets.",
			MinInputField: "company",
			RiskLevelNote: "Low: public business context only.",
		},
	}

	docs := map[string]string{
		models.ModuleEmailValidate: "Runs AfterShip/email-verifier with the SMTP probe disabled. " +
			"Results are namespaced under the `email_validate` key. Audit records include the email checked, tool version, status and legal basis.",
		models.ModulePhoneValidate: "Runs the offline libphonenumber-based local scanner. " +
			"If NUMVERIFY_API_KEY is set an optional carrier lookup is attempted. Phone numbers are redacted in audit logs.",
		models.ModuleDomainIntel:     "Not wired in control-plane v1. Will run a Go reimplementation of web-check plus a theHarvester subprocess in a future PR.",
		models.ModuleSocialFootprint: "Not wired in control-plane v1. Will run the Maigret Python wrapper with a curated platform allow-list and an in-process rate limiter.",
		models.ModuleExtraction:      "Planned for a future PR.",
		models.ModuleCompanyEnrich:   "Planned for a future PR.",
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
