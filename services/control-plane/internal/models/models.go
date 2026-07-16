// Package models defines the domain types shared by the control-plane HTTP API,
// storage layer, and module runner. Field names and JSON tags mirror the frontend
// API contract in docs/frontend/api-contracts.md.
package models

import (
	"time"
)

// PipelineStage values. A lead progresses raw -> enriched -> validated -> crm_ready.
const (
	StageRaw       = "raw"
	StageEnriched  = "enriched"
	StageValidated = "validated"
	StageCrmReady  = "crm_ready"
)

// RiskLevel values.
const (
	RiskLow     = "low"
	RiskMedium  = "medium"
	RiskHigh    = "high"
	RiskUnknown = "unknown"
)

// Module names.
const (
	ModuleEmailValidate   = "email-validate"
	ModulePhoneValidate   = "phone-validate"
	ModuleDomainIntel     = "domain-intel"
	ModuleSocialFootprint = "social-footprint"
	ModuleExtraction      = "extraction"
	ModuleCompanyEnrich   = "company-enrich"
)

// RawLead is the user-supplied input for a new lead. All fields are optional
// except where noted in the contract.
type RawLead struct {
	Name          string `json:"name"`
	Email         string `json:"email"`
	Phone         string `json:"phone"`
	Company       string `json:"company"`
	Domain        string `json:"domain"`
	SourceID      string `json:"source_id"`
	PermissionRef string `json:"permission_ref"`
}

// Lead is a lead record including pipeline state and namespaced module results.
// Results is intentionally map[string]any so module outputs stay namespaced and
// the storage layer can marshal them as JSONB.
type Lead struct {
	ID            string         `json:"id"`
	Name          string         `json:"name"`
	Email         string         `json:"email"`
	Phone         string         `json:"phone"`
	Company       string         `json:"company"`
	Domain        string         `json:"domain"`
	SourceID      string         `json:"source_id"`
	PermissionRef string         `json:"permission_ref"`
	Stage         string         `json:"stage"`
	RiskLevel     string         `json:"risk_level"`
	RiskScore     *float64       `json:"risk_score,omitempty"`
	Results       map[string]any `json:"results"`
	AuditEvents   []AuditEvent   `json:"audit_events"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
}

// Subject identifies the target of an audit line. Only one field is set per event
// and PII is redacted before storage.
type Subject struct {
	Email         string `json:"email,omitempty"`
	Domain        string `json:"domain,omitempty"`
	PhoneRedacted string `json:"phone_redacted,omitempty"`
	Handle        string `json:"handle,omitempty"`
}

// AuditEvent is one structured audit line. raw_stderr_json preserves the
// module's stderr/audit payload verbatim.
type AuditEvent struct {
	ID            string    `json:"id"`
	LeadID        string    `json:"lead_id"`
	RunID         *string   `json:"run_id,omitempty"`
	Module        string    `json:"module"`
	Tool          string    `json:"tool"`
	CheckedAt     time.Time `json:"checked_at"`
	Status        string    `json:"status"`
	LegalBasis    string    `json:"legal_basis"`
	Subject       Subject   `json:"subject"`
	RawStderrJSON string    `json:"raw_stderr_json,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
}

// PipelineRun records a single-lead or batch execution.
type PipelineRun struct {
	ID              string     `json:"id"`
	Type            string     `json:"type"`
	Status          string     `json:"status"`
	StartedAt       time.Time  `json:"started_at"`
	FinishedAt      *time.Time `json:"finished_at,omitempty"`
	LeadIDs         []string   `json:"lead_ids"`
	ModulesExecuted []string   `json:"modules_executed"`
	AuditEventIDs   []string   `json:"audit_event_ids"`
	LegalBasis      string     `json:"legal_basis"`
	PermissionRefs  []string   `json:"permission_refs"`
	Error           string     `json:"error,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
}

// ComplianceRule is one hard-coded governance rule.
type ComplianceRule struct {
	ID      int    `json:"id"`
	Title   string `json:"title"`
	Summary string `json:"summary"`
}

// ComplianceRiskRule maps a risk category to a risk level note.
type ComplianceRiskRule struct {
	Category  string `json:"category"`
	RiskLevel string `json:"risk_level"`
	Notes     string `json:"notes"`
}

// ChecklistItem is a compliance checklist entry.
type ChecklistItem struct {
	ID    string `json:"id"`
	Label string `json:"label"`
}

// ComplianceSummary is returned by GET /api/compliance/summary.
type ComplianceSummary struct {
	HardRules  []ComplianceRule     `json:"hard_rules"`
	RiskTable  []ComplianceRiskRule `json:"risk_table"`
	Checklist  []ChecklistItem      `json:"checklist"`
	Exclusions []string             `json:"exclusions"`
}

// LegalBasisGDPR is the default legal basis used when none is supplied.
const LegalBasisGDPR = "GDPR Art.6(1)(f) legitimate-interest"

// ModuleResult is the runner's output for one module on one lead. The Result
// value must be JSON-serializable; AuditEvents carry the module's audit trail.
type ModuleResult struct {
	Key         string
	Result      any
	AuditEvents []AuditEvent
}

// RunModulesRequest is the body of POST /api/leads/{id}/run.
type RunModulesRequest struct {
	Modules       []string `json:"modules"`
	PermissionRef string   `json:"permission_ref"`
	LegalBasis    string   `json:"legal_basis"`
}

// PipelineRunRequest is the body of POST /api/pipelines/run.
type PipelineRunRequest struct {
	LeadIDs       []string `json:"lead_ids"`
	Modules       []string `json:"modules"`
	PermissionRef string   `json:"permission_ref"`
	LegalBasis    string   `json:"legal_basis"`
}

// ModuleInfo describes a module for GET /api/modules.
type ModuleInfo struct {
	Name          string   `json:"name"`
	DisplayName   string   `json:"display_name"`
	Category      string   `json:"category"`
	DevStatus     string   `json:"dev_status"`
	NamespacedKey string   `json:"namespaced_key"`
	BackingTools  []string `json:"backing_tools"`
	Description   string   `json:"description"`
	MinInputField string   `json:"min_input_field"`
	RiskLevelNote string   `json:"risk_level_note"`
}

// ModuleDetail extends ModuleInfo with optional documentation.
type ModuleDetail struct {
	ModuleInfo
	Docs string `json:"docs,omitempty"`
}

// LeadSearchParams carries the accepted query parameters for GET /api/leads.
type LeadSearchParams struct {
	Stage        string
	Risk         string
	ModuleStatus string
	Q            string
	Page         int
	PageSize     int
}

// AuditSearchParams carries query parameters for GET /api/audit.
type AuditSearchParams struct {
	Module   string
	Status   string
	Page     int
	PageSize int
}
