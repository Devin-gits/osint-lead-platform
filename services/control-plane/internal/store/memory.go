package store

import (
	"context"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/Moyeil-73/osint-lead-platform/services/control-plane/internal/models"
)

// MemoryStore is an in-memory Store implementation for tests and local dev.
type MemoryStore struct {
	mu    sync.RWMutex
	leads map[string]models.Lead
	audit map[string]models.AuditEvent
	runs  map[string]models.PipelineRun
}

// NewMemoryStore returns a fresh, empty MemoryStore.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		leads: make(map[string]models.Lead),
		audit: make(map[string]models.AuditEvent),
		runs:  make(map[string]models.PipelineRun),
	}
}

func (s *MemoryStore) CreateLead(_ context.Context, lead models.Lead) (models.Lead, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if lead.ID == "" {
		return lead, ErrInvalid
	}
	if _, exists := s.leads[lead.ID]; exists {
		return lead, ErrInvalid
	}
	if lead.Stage == "" {
		lead.Stage = models.StageRaw
	}
	if lead.RiskLevel == "" {
		lead.RiskLevel = models.RiskNA
	}
	if lead.Results == nil {
		lead.Results = map[string]any{}
	}
	t := now()
	lead.CreatedAt = t
	lead.UpdatedAt = t
	s.leads[lead.ID] = lead
	return lead, nil
}

func (s *MemoryStore) ListLeads(_ context.Context, params models.LeadSearchParams) ([]models.Lead, int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	all := make([]models.Lead, 0, len(s.leads))
	for _, l := range s.leads {
		all = append(all, l)
	}
	sort.Slice(all, func(i, j int) bool {
		return all[i].CreatedAt.After(all[j].CreatedAt)
	})

	filtered, total := filterLeads(all, params)
	return filtered, total, nil
}

func (s *MemoryStore) GetLead(_ context.Context, id string) (models.Lead, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	l, ok := s.leads[id]
	if !ok {
		return models.Lead{}, ErrNotFound
	}
	return l, nil
}

func (s *MemoryStore) UpdateLead(_ context.Context, lead models.Lead) (models.Lead, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.leads[lead.ID]; !ok {
		return models.Lead{}, ErrNotFound
	}
	lead.UpdatedAt = now()
	s.leads[lead.ID] = lead
	return lead, nil
}

func (s *MemoryStore) CreateAuditEvent(_ context.Context, event models.AuditEvent) (models.AuditEvent, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if event.ID == "" {
		return event, ErrInvalid
	}
	event.CreatedAt = now()
	s.audit[event.ID] = event
	return event, nil
}

func (s *MemoryStore) ListAuditEvents(_ context.Context, params models.AuditSearchParams) ([]models.AuditEvent, int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	page, pageSize := normalizePagination(params.Page, params.PageSize)
	filtered := make([]models.AuditEvent, 0, len(s.audit))
	for _, e := range s.audit {
		if params.Module != "" && !strings.EqualFold(e.Module, params.Module) {
			continue
		}
		if params.Status != "" && !strings.EqualFold(e.Status, params.Status) {
			continue
		}
		filtered = append(filtered, e)
	}
	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].CreatedAt.After(filtered[j].CreatedAt)
	})

	total := len(filtered)
	start := (page - 1) * pageSize
	if start > total {
		start = total
	}
	end := start + pageSize
	if end > total {
		end = total
	}
	return filtered[start:end], total, nil
}

func (s *MemoryStore) CreatePipelineRun(_ context.Context, run models.PipelineRun) (models.PipelineRun, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if run.ID == "" {
		return run, ErrInvalid
	}
	run.CreatedAt = now()
	s.runs[run.ID] = run
	return run, nil
}

func (s *MemoryStore) ListPipelineRuns(_ context.Context, params models.AuditSearchParams) ([]models.PipelineRun, int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	page, pageSize := normalizePagination(params.Page, params.PageSize)
	all := make([]models.PipelineRun, 0, len(s.runs))
	for _, r := range s.runs {
		all = append(all, r)
	}
	sort.Slice(all, func(i, j int) bool {
		return all[i].StartedAt.After(all[j].StartedAt)
	})
	total := len(all)
	start := (page - 1) * pageSize
	if start > total {
		start = total
	}
	end := start + pageSize
	if end > total {
		end = total
	}
	return all[start:end], total, nil
}

func (s *MemoryStore) GetPipelineRun(_ context.Context, id string) (models.PipelineRun, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	r, ok := s.runs[id]
	if !ok {
		return models.PipelineRun{}, ErrNotFound
	}
	return r, nil
}

func (s *MemoryStore) UpdatePipelineRun(_ context.Context, run models.PipelineRun) (models.PipelineRun, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.runs[run.ID]; !ok {
		return models.PipelineRun{}, ErrNotFound
	}
	s.runs[run.ID] = run
	return run, nil
}

func (s *MemoryStore) ComplianceSummary(_ context.Context) (models.ComplianceSummary, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	summary := models.ComplianceSummary{
		LeadsByStage: make(map[string]int),
		LeadsByRisk:  make(map[string]int),
	}
	for _, l := range s.leads {
		summary.TotalLeads++
		summary.LeadsByStage[l.Stage]++
		summary.LeadsByRisk[l.RiskLevel]++
	}
	for _, e := range s.audit {
		summary.TotalAuditEvents++
		if e.CreatedAt.After(time.Now().UTC().Add(-24 * time.Hour)) {
			summary.Last24hAuditEvents++
		}
	}
	return summary, nil
}
