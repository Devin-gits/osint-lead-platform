// Package store abstracts persistence for leads, audit events, and pipeline runs.
// A memory implementation is provided for tests; the production default uses
// Postgres via pgx.
package store

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/Moyeil-73/osint-lead-platform/services/control-plane/internal/models"
)

// Common errors.
var (
	ErrNotFound = errors.New("resource not found")
	ErrInvalid  = errors.New("invalid input")
)

// Store is the persistence contract used by the HTTP API and runner.
type Store interface {
	CreateLead(ctx context.Context, lead models.Lead) (models.Lead, error)
	ListLeads(ctx context.Context, params models.LeadSearchParams) ([]models.Lead, int, error)
	GetLead(ctx context.Context, id string) (models.Lead, error)
	UpdateLead(ctx context.Context, lead models.Lead) (models.Lead, error)

	CreateAuditEvent(ctx context.Context, event models.AuditEvent) (models.AuditEvent, error)
	ListAuditEvents(ctx context.Context, params models.AuditSearchParams) ([]models.AuditEvent, int, error)
	ListAuditEventsByLead(ctx context.Context, leadID string) ([]models.AuditEvent, error)

	CreatePipelineRun(ctx context.Context, run models.PipelineRun) (models.PipelineRun, error)
	ListPipelineRuns(ctx context.Context, params models.AuditSearchParams) ([]models.PipelineRun, int, error)
	GetPipelineRun(ctx context.Context, id string) (models.PipelineRun, error)
	UpdatePipelineRun(ctx context.Context, run models.PipelineRun) (models.PipelineRun, error)

	ComplianceSummary(ctx context.Context) (models.ComplianceSummary, error)
}

// filterLeads performs in-memory filtering and pagination. PostgresStore uses it
// after loading candidate rows so both implementations behave identically.
func filterLeads(leads []models.Lead, params models.LeadSearchParams) ([]models.Lead, int) {
	if params.Page < 1 {
		params.Page = 1
	}
	if params.PageSize < 1 {
		params.PageSize = 25
	}
	if params.PageSize > 100 {
		params.PageSize = 100
	}

	filtered := make([]models.Lead, 0, len(leads))
	for _, l := range leads {
		if !matchString(params.Stage, l.Stage) {
			continue
		}
		if !matchString(params.Risk, l.RiskLevel) {
			continue
		}
		if params.ModuleStatus != "" && !matchModuleStatus(l, params.ModuleStatus) {
			continue
		}
		if params.Q != "" && !matchFreeText(l, strings.ToLower(params.Q)) {
			continue
		}
		filtered = append(filtered, l)
	}

	total := len(filtered)
	start := (params.Page - 1) * params.PageSize
	if start > total {
		start = total
	}
	end := start + params.PageSize
	if end > total {
		end = total
	}
	return filtered[start:end], total
}

func matchString(want, got string) bool {
	return want == "" || strings.EqualFold(want, got)
}

func matchFreeText(l models.Lead, q string) bool {
	return strings.Contains(strings.ToLower(l.Name), q) ||
		strings.Contains(strings.ToLower(l.Email), q) ||
		strings.Contains(strings.ToLower(l.Company), q) ||
		strings.Contains(strings.ToLower(l.Domain), q) ||
		strings.Contains(strings.ToLower(l.URL), q)
}

// matchModuleStatus reports whether any of the namespaced module result
// blocks has a status equal to want. A missing key is treated as "not_run".
func matchModuleStatus(l models.Lead, want string) bool {
	keys := []string{"email_validate", "phone_validate", "domain_intel", "social_footprint", "extraction", "company_enrich"}
	for _, key := range keys {
		status := moduleResultStatus(l.Results, key)
		if strings.EqualFold(status, want) {
			return true
		}
	}
	return false
}

func moduleResultStatus(results map[string]any, key string) string {
	v, ok := results[key]
	if !ok {
		return "not_run"
	}
	m, ok := v.(map[string]any)
	if !ok {
		return "not_run"
	}
	s, _ := m["status"].(string)
	if s == "" {
		return "not_run"
	}
	return s
}

// normalizePagination ensures sane defaults.
func normalizePagination(page, pageSize int) (int, int) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 25
	}
	if pageSize > 100 {
		pageSize = 100
	}
	return page, pageSize
}

// now returns UTC time; isolated to avoid time.Since in package-level vars.
func now() time.Time { return time.Now().UTC() }
