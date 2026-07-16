package store

import (
	"github.com/Moyeil-73/osint-lead-platform/services/control-plane/internal/models"
)

// staticComplianceSummary returns the governance content expected by the
// frontend compliance page. It is intentionally static and derived from
// docs/compliance.md so the API can return the same copy without re-parsing.
func staticComplianceSummary() models.ComplianceSummary {
	return models.ComplianceSummary{
		HardRules: []models.ComplianceRule{
			{
				ID:      1,
				Title:   "No non-consensual personal surveillance",
				Summary: "Do not perform covert or non-consensual surveillance of individuals. Checks are limited to public business context and known identifiers provided by the lead.",
			},
			{
				ID:      2,
				Title:   "Respect third-party ToS",
				Summary: "Automated scraping or bypassing platform terms of service is prohibited. Use only allowed APIs, public DNS/WHOIS data, and subprocess tools with explicit legal basis.",
			},
			{
				ID:      3,
				Title:   "Rate-limit and document breach-checking",
				Summary: "Any third-party lookup must be rate-limited and logged with legal basis. Bulk breach/leak signals are not surfaced in sales views.",
			},
			{
				ID:      4,
				Title:   "Log the legal basis and source permission reference",
				Summary: "Every lead and every audit event must record a permission reference and a GDPR legal basis, defaulting to Art.6(1)(f) legitimate interest.",
			},
			{
				ID:      5,
				Title:   "Data retention",
				Summary: "Enrichment results must be retained only for the configured retention window and deleted or exported to CRM before expiry.",
			},
		},
		RiskTable: []models.ComplianceRiskRule{
			{
				Category:  "Email verification",
				RiskLevel: "Low",
				Notes:     "No email is sent; only syntax, MX and public reputation checks are used. SMTP deliverability probes are disabled.",
			},
			{
				Category:  "Phone validation",
				RiskLevel: "Low-Medium",
				Notes:     "Offline libphonenumber parsing is low risk. Optional carrier lookups are rate-limited and phone numbers are redacted in audit logs.",
			},
			{
				Category:  "Domain intelligence",
				RiskLevel: "Low",
				Notes:     "DNS, TLS, WHOIS and public-subdomain checks are public business context.",
			},
			{
				Category:  "Social footprint",
				RiskLevel: "Medium",
				Notes:     "Only handle presence or absence is recorded from a curated platform allow-list. No scraped profile fields are stored.",
			},
		},
		Checklist: []models.ChecklistItem{
			{ID: "permission", Label: "Permission ref recorded for every lead"},
			{ID: "legal_basis", Label: "Legal basis confirmed before enrichment"},
			{ID: "rate_limit", Label: "Rate limits respected for third-party lookups"},
			{ID: "retention", Label: "Retention window defined for this run"},
			{ID: "review", Label: "Results reviewed before CRM export"},
		},
		Exclusions: []string{
			"LinkedIn scraping",
			"Reverse-image / deep account discovery",
			"Bulk breach/leak signals in sales views",
		},
	}
}
