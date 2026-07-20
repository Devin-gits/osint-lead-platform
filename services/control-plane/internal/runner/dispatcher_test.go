package runner

import (
	"context"
	"errors"
	"testing"

	"github.com/Moyeil-73/osint-lead-platform/services/control-plane/internal/models"
	"github.com/Moyeil-73/osint-lead-platform/services/control-plane/internal/store"
	"github.com/Moyeil-73/osint-lead-platform/services/control-plane/internal/util"
)

func TestRunnerSubmitSinglePublishesRunIDAndReleasesAfterCompletion(t *testing.T) {
	ctx := context.Background()
	st := store.NewMemoryStore()
	r := New(st, 0)
	r.ctx, r.cancel = context.WithCancel(ctx)
	r.jobs = make(chan job, 1)
	r.activeRuns = make(map[string]string)
	r.started = true
	t.Cleanup(r.cancel)

	lead := models.Lead{ID: util.NewID(), Email: "support@github.com"}
	if _, err := st.CreateLead(ctx, lead); err != nil {
		t.Fatalf("create lead: %v", err)
	}

	run, err := r.SubmitSingle(ctx, lead.ID, models.RunModulesRequest{Modules: []string{"email-validate"}})
	if err != nil {
		t.Fatalf("submit single: %v", err)
	}
	if active := r.ActiveRun(lead.ID); active != run.ID {
		t.Fatalf("expected active run %q before worker completion, got %q", run.ID, active)
	}

	if _, err := r.SubmitSingle(ctx, lead.ID, models.RunModulesRequest{Modules: []string{"email-validate"}}); !errors.Is(err, ErrRunInProgress) {
		t.Fatalf("expected ErrRunInProgress, got %v", err)
	}

	r.processJob(ctx, <-r.jobs)
	if active := r.ActiveRun(lead.ID); active != "" {
		t.Fatalf("expected no active run after completion, got %q", active)
	}
}
