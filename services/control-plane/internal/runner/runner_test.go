package runner

import (
	"context"
	"os/exec"
	"testing"
	"time"

	"github.com/Moyeil-73/osint-lead-platform/services/control-plane/internal/models"
	"github.com/Moyeil-73/osint-lead-platform/services/control-plane/internal/store"
	"github.com/Moyeil-73/osint-lead-platform/services/control-plane/internal/util"
)

func TestRunner_RunSingleEmail(t *testing.T) {
	ctx := context.Background()
	st := store.NewMemoryStore()
	r := New(st, 0)

	lead := models.Lead{ID: util.NewID(), Email: "support@github.com"}
	if _, err := st.CreateLead(ctx, lead); err != nil {
		t.Fatalf("create lead: %v", err)
	}

	updated, err := r.RunSingle(ctx, lead.ID, models.RunModulesRequest{
		Modules: []string{"email-validate"},
	})
	if err != nil {
		t.Fatalf("run modules: %v", err)
	}

	ev, ok := updated.Results["email_validate"].(map[string]any)
	if !ok {
		t.Fatalf("expected email_validate result, got %T", updated.Results["email_validate"])
	}
	status, _ := ev["status"].(string)
	if status != "ok" {
		t.Fatalf("expected status ok, got %s", status)
	}
	if updated.Stage != models.StageValidated {
		t.Fatalf("expected stage validated, got %s", updated.Stage)
	}

	// Audit event should be persisted.
	events, _, err := st.ListAuditEvents(ctx, models.AuditSearchParams{})
	if err != nil {
		t.Fatalf("list audit: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 audit event, got %d", len(events))
	}
	if events[0].Module != "email-validate" {
		t.Fatalf("expected module email-validate, got %s", events[0].Module)
	}
	if events[0].LegalBasis != models.LegalBasisGDPR {
		t.Fatalf("expected GDPR legal basis, got %s", events[0].LegalBasis)
	}

	// Pipeline run should be completed.
	runs, _, err := st.ListPipelineRuns(ctx, models.AuditSearchParams{})
	if err != nil {
		t.Fatalf("list runs: %v", err)
	}
	if len(runs) != 1 || runs[0].Status != "completed" {
		t.Fatalf("expected 1 completed run, got %+v", runs)
	}
}

func TestRunner_RunSingleDomainIntelMissingDomain(t *testing.T) {
	ctx := context.Background()
	st := store.NewMemoryStore()
	r := New(st, 0)

	lead := models.Lead{ID: util.NewID(), Email: "support@github.com"}
	if _, err := st.CreateLead(ctx, lead); err != nil {
		t.Fatalf("create lead: %v", err)
	}

	updated, err := r.RunSingle(ctx, lead.ID, models.RunModulesRequest{
		Modules: []string{"domain-intel"},
	})
	if err != nil {
		t.Fatalf("run modules: %v", err)
	}

	ev, ok := updated.Results["domain_intel"].(map[string]any)
	if !ok {
		t.Fatalf("expected domain_intel result, got %T", updated.Results["domain_intel"])
	}
	if ev["status"] != "skipped" {
		t.Fatalf("expected skipped status, got %v", ev["status"])
	}
	if ev["reason"] != "missing domain" {
		t.Fatalf("expected missing domain reason, got %v", ev["reason"])
	}
	if updated.Stage != models.StageRaw {
		t.Fatalf("expected stage to remain raw on pure skip, got %s", updated.Stage)
	}
}

func TestRunner_RunSingleDomainIntel(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping network test in short mode")
	}
	ctx := context.Background()
	st := store.NewMemoryStore()
	r := New(st, 0)

	lead := models.Lead{ID: util.NewID(), Email: "support@github.com", Domain: "github.com"}
	if _, err := st.CreateLead(ctx, lead); err != nil {
		t.Fatalf("create lead: %v", err)
	}

	updated, err := r.RunSingle(ctx, lead.ID, models.RunModulesRequest{
		Modules: []string{"domain-intel"},
	})
	if err != nil {
		t.Fatalf("run modules: %v", err)
	}

	ev, ok := updated.Results["domain_intel"].(map[string]any)
	if !ok {
		t.Fatalf("expected domain_intel result, got %T", updated.Results["domain_intel"])
	}
	status, _ := ev["status"].(string)
	if status == "skipped" {
		t.Fatalf("expected domain_intel not skipped, got %v", status)
	}
	if ev["web_check"] == nil {
		t.Fatalf("expected web_check sub-result")
	}
	if updated.Stage != models.StageEnriched {
		t.Fatalf("expected stage enriched when domain-intel ok, got %s", updated.Stage)
	}

	// Audit events should include legal basis and domain subject.
	events, _, err := st.ListAuditEvents(ctx, models.AuditSearchParams{})
	if err != nil {
		t.Fatalf("list audit: %v", err)
	}
	found := false
	for _, e := range events {
		if e.Module == "domain-intel" && e.LegalBasis == models.LegalBasisGDPR && e.Subject.Domain == "github.com" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected domain-intel audit event with legal basis and domain subject, got %+v", events)
	}
}

func TestRunner_RunBatch(t *testing.T) {
	ctx := context.Background()
	st := store.NewMemoryStore()
	r := New(st, 0)

	lead := models.Lead{ID: util.NewID(), Email: "support@github.com"}
	if _, err := st.CreateLead(ctx, lead); err != nil {
		t.Fatalf("create lead: %v", err)
	}

	run, err := r.RunBatch(ctx, models.PipelineRunRequest{
		LeadIDs: []string{lead.ID},
		Modules: []string{"email-validate"},
	})
	if err != nil {
		t.Fatalf("run batch: %v", err)
	}
	if run.Status != "completed" {
		t.Fatalf("expected run completed, got %s", run.Status)
	}
}

func TestRunner_RunSingleSocialFootprintMissingHandle(t *testing.T) {
	ctx := context.Background()
	st := store.NewMemoryStore()
	r := New(st, 5*time.Second)

	lead := models.Lead{ID: util.NewID(), Name: "No Email No Domain"}
	if _, err := st.CreateLead(ctx, lead); err != nil {
		t.Fatalf("create lead: %v", err)
	}

	updated, err := r.RunSingle(ctx, lead.ID, models.RunModulesRequest{
		Modules: []string{"social-footprint"},
	})
	if err != nil {
		t.Fatalf("run modules: %v", err)
	}

	ev, ok := updated.Results["social_footprint"].(map[string]any)
	if !ok {
		t.Fatalf("expected social_footprint result, got %T", updated.Results["social_footprint"])
	}
	if ev["status"] != "skipped" {
		t.Fatalf("expected skipped status, got %v", ev["status"])
	}
	if updated.Stage != models.StageRaw {
		t.Fatalf("expected stage raw on skip, got %s", updated.Stage)
	}

	events, _, err := st.ListAuditEvents(ctx, models.AuditSearchParams{})
	if err != nil {
		t.Fatalf("list audit: %v", err)
	}
	found := false
	for _, e := range events {
		if e.Module == "social-footprint" && e.LegalBasis == models.LegalBasisGDPR {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected social-footprint audit event with legal basis, got %+v", events)
	}
}

func TestRunner_RunSingleSocialFootprint(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping network/subprocess social-footprint test in short mode")
	}

	ctx := context.Background()
	st := store.NewMemoryStore()
	r := New(st, 10*time.Second)

	lead := models.Lead{ID: util.NewID(), Email: "jane.smith@acme.com"}
	if _, err := st.CreateLead(ctx, lead); err != nil {
		t.Fatalf("create lead: %v", err)
	}

	updated, err := r.RunSingle(ctx, lead.ID, models.RunModulesRequest{
		Modules: []string{"social-footprint"},
	})
	if err != nil {
		t.Fatalf("run modules: %v", err)
	}

	ev, ok := updated.Results["social_footprint"].(map[string]any)
	if !ok {
		t.Fatalf("expected social_footprint result, got %T", updated.Results["social_footprint"])
	}
	status, _ := ev["status"].(string)
	if status != "ok" && status != "unknown" {
		t.Fatalf("expected social_footprint ok or unknown, got %v", status)
	}

	handles, _ := ev["handles_checked"].([]any)
	if len(handles) == 0 {
		t.Fatalf("expected handles_checked to be non-empty")
	}

	if updated.Stage != models.StageValidated {
		t.Fatalf("expected stage validated when social-footprint reports ok, got %s", updated.Stage)
	}

	events, _, err := st.ListAuditEvents(ctx, models.AuditSearchParams{})
	if err != nil {
		t.Fatalf("list audit: %v", err)
	}
	found := false
	for _, e := range events {
		if e.Module == "social-footprint" && e.LegalBasis == models.LegalBasisGDPR && e.Subject.Handle != "" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected social-footprint audit event with handle subject and legal basis, got %+v", events)
	}
}

func TestRunner_RunSingleSocialFootprintUsesDomainIntel(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping subprocess social-footprint test in short mode")
	}
	if _, err := exec.LookPath("python3"); err != nil {
		t.Skip("python3 not available; skipping live Maigret subprocess test")
	}

	ctx := context.Background()
	st := store.NewMemoryStore()
	r := New(st, 15*time.Second)

	lead := models.Lead{
		ID:    util.NewID(),
		Email: "admin@acme.com",
		Results: map[string]any{
			"domain_intel": map[string]any{
				"harvester": map[string]any{
					"emails": []any{"jane.smith@acme.com"},
					"hosts":  []any{map[string]any{"host": "jane.smith.acme.com"}},
				},
			},
		},
	}
	if _, err := st.CreateLead(ctx, lead); err != nil {
		t.Fatalf("create lead: %v", err)
	}

	updated, err := r.RunSingle(ctx, lead.ID, models.RunModulesRequest{
		Modules: []string{"social-footprint"},
	})
	if err != nil {
		t.Fatalf("run modules: %v", err)
	}

	ev, ok := updated.Results["social_footprint"].(map[string]any)
	if !ok {
		t.Fatalf("expected social_footprint result, got %T", updated.Results["social_footprint"])
	}
	handles, _ := ev["handles_checked"].([]any)
	if len(handles) < 2 {
		t.Fatalf("expected at least 2 handle candidates from email + domain-intel harvester, got %d", len(handles))
	}
}
