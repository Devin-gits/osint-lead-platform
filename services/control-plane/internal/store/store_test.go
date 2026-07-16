package store

import (
	"context"
	"testing"
	"time"

	"github.com/Moyeil-73/osint-lead-platform/services/control-plane/internal/models"
	"github.com/Moyeil-73/osint-lead-platform/services/control-plane/internal/util"
)

func TestMemoryStore_LeadLifecycle(t *testing.T) {
	ctx := context.Background()
	s := NewMemoryStore()

	lead := models.Lead{ID: util.NewID(), Name: "Acme", Email: "support@github.com"}
	created, err := s.CreateLead(ctx, lead)
	if err != nil {
		t.Fatalf("create lead: %v", err)
	}
	if created.Stage != models.StageRaw {
		t.Fatalf("expected stage raw, got %s", created.Stage)
	}

	got, err := s.GetLead(ctx, lead.ID)
	if err != nil {
		t.Fatalf("get lead: %v", err)
	}
	if got.Email != lead.Email {
		t.Fatalf("email mismatch")
	}

	got.RiskLevel = models.RiskLow
	got.Results["email_validate"] = map[string]any{"status": "ok"}
	updated, err := s.UpdateLead(ctx, got)
	if err != nil {
		t.Fatalf("update lead: %v", err)
	}
	if updated.RiskLevel != models.RiskLow {
		t.Fatalf("risk not updated")
	}

	leads, total, err := s.ListLeads(ctx, models.LeadSearchParams{PageSize: 10})
	if err != nil {
		t.Fatalf("list leads: %v", err)
	}
	if total != 1 || len(leads) != 1 {
		t.Fatalf("expected 1 lead, got %d / %d", len(leads), total)
	}

	_, _, err = s.ListLeads(ctx, models.LeadSearchParams{ModuleStatus: "ok"})
	if err != nil {
		t.Fatalf("filter by module status: %v", err)
	}
}

func TestMemoryStore_AuditAndRuns(t *testing.T) {
	ctx := context.Background()
	s := NewMemoryStore()

	lead := models.Lead{ID: util.NewID()}
	if _, err := s.CreateLead(ctx, lead); err != nil {
		t.Fatalf("create lead: %v", err)
	}

	evt := models.AuditEvent{
		ID:         util.NewID(),
		LeadID:     lead.ID,
		Module:     "email-validate",
		Tool:       "test",
		CheckedAt:  time.Now().UTC(),
		Status:     "ok",
		LegalBasis: models.LegalBasisGDPR,
	}
	if _, err := s.CreateAuditEvent(ctx, evt); err != nil {
		t.Fatalf("create audit: %v", err)
	}

	run := models.PipelineRun{
		ID:         util.NewID(),
		Type:       "single",
		Status:     "completed",
		StartedAt:  time.Now().UTC(),
		LeadIDs:    []string{lead.ID},
		LegalBasis: models.LegalBasisGDPR,
	}
	if _, err := s.CreatePipelineRun(ctx, run); err != nil {
		t.Fatalf("create run: %v", err)
	}

	runs, total, err := s.ListPipelineRuns(ctx, models.AuditSearchParams{PageSize: 10})
	if err != nil {
		t.Fatalf("list runs: %v", err)
	}
	if total != 1 || len(runs) != 1 {
		t.Fatalf("expected 1 run, got %d / %d", len(runs), total)
	}
}
