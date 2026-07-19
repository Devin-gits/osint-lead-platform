package runner

import (
	"context"
	"fmt"
	"time"

	"github.com/Moyeil-73/osint-lead-platform/services/control-plane/internal/models"
	"github.com/Moyeil-73/osint-lead-platform/services/control-plane/internal/store"
	"github.com/Moyeil-73/osint-lead-platform/services/control-plane/internal/util"
)

// job is a queued module execution.
type job struct {
	runID    string
	leadID   string // empty for batch jobs
	single   *models.RunModulesRequest
	batch    *models.PipelineRunRequest
}

// Start launches the worker goroutine pool. It must be called before any job is
// submitted; it is safe to call multiple times (subsequent calls are no-ops).
func (r *Runner) Start(ctx context.Context, workers int) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.started {
		return
	}

	if workers < 1 {
		workers = 2
	}

	r.ctx, r.cancel = context.WithCancel(ctx)
	r.jobs = make(chan job, 100)
	r.activeRuns = make(map[string]string)
	r.started = true

	for i := 0; i < workers; i++ {
		r.wg.Add(1)
		go r.worker()
	}
}

// Stop signals workers to finish and waits for them. The Runner cannot be
// restarted after Stop.
func (r *Runner) Stop() {
	r.mu.Lock()
	if !r.started {
		r.mu.Unlock()
		return
	}
	r.cancel()
	close(r.jobs)
	r.started = false
	r.mu.Unlock()

	r.wg.Wait()
}

// SubmitSingle enqueues a single-lead module run. It returns a PipelineRun with
// status "queued". Only one active run is allowed per lead; a second request
// for the same lead while a run is queued or running returns ErrRunInProgress.
func (r *Runner) SubmitSingle(ctx context.Context, leadID string, req models.RunModulesRequest) (models.PipelineRun, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if !r.started {
		return models.PipelineRun{}, fmt.Errorf("runner not started")
	}

	lead, err := r.store.GetLead(ctx, leadID)
	if err != nil {
		return models.PipelineRun{}, err
	}

	legalBasis := req.LegalBasis
	if legalBasis == "" {
		legalBasis = models.LegalBasisGDPR
	}
	permissionRef := req.PermissionRef
	if permissionRef == "" {
		permissionRef = lead.PermissionRef
	}

	if err := r.reserveActive(leadID); err != nil {
		return models.PipelineRun{}, err
	}

	run := models.PipelineRun{
		ID:              util.NewID(),
		Type:            "single",
		Status:          "queued",
		StartedAt:       time.Now().UTC(),
		LeadIDs:         []string{leadID},
		ModulesExecuted: req.Modules,
		LegalBasis:      legalBasis,
		PermissionRefs:  uniqueStrings(append([]string{}, permissionRef)),
	}

	if _, err := r.store.CreatePipelineRun(ctx, run); err != nil {
		r.releaseActive(leadID)
		return models.PipelineRun{}, fmt.Errorf("create run: %w", err)
	}

	select {
	case r.jobs <- job{runID: run.ID, leadID: leadID, single: &req}:
		return run, nil
	case <-r.ctx.Done():
		r.releaseActive(leadID)
		return models.PipelineRun{}, fmt.Errorf("runner shutting down")
	default:
		r.releaseActive(leadID)
		return models.PipelineRun{}, fmt.Errorf("job queue full")
	}
}

// SubmitBatch enqueues a batch module run. Only one active run is allowed per
// lead; if any lead in the batch is already queued or running, it returns
// ErrRunInProgress.
func (r *Runner) SubmitBatch(ctx context.Context, req models.PipelineRunRequest) (models.PipelineRun, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if !r.started {
		return models.PipelineRun{}, fmt.Errorf("runner not started")
	}

	if len(req.LeadIDs) == 0 {
		return models.PipelineRun{}, store.ErrInvalid
	}
	if len(req.Modules) == 0 {
		return models.PipelineRun{}, store.ErrInvalid
	}

	legalBasis := req.LegalBasis
	if legalBasis == "" {
		legalBasis = models.LegalBasisGDPR
	}

	for _, leadID := range req.LeadIDs {
		if err := r.reserveActive(leadID); err != nil {
			for _, lid := range req.LeadIDs {
				if lid == leadID {
					break
				}
				r.releaseActive(lid)
			}
			return models.PipelineRun{}, err
		}
	}

	run := models.PipelineRun{
		ID:              util.NewID(),
		Type:            "batch",
		Status:          "queued",
		StartedAt:       time.Now().UTC(),
		LeadIDs:         req.LeadIDs,
		ModulesExecuted: req.Modules,
		LegalBasis:      legalBasis,
		PermissionRefs:  uniqueStrings(append([]string{}, req.PermissionRef)),
	}
	if _, err := r.store.CreatePipelineRun(ctx, run); err != nil {
		r.releaseActiveBatch(req.LeadIDs)
		return models.PipelineRun{}, fmt.Errorf("create run: %w", err)
	}

	select {
	case r.jobs <- job{runID: run.ID, batch: &req}:
		return run, nil
	case <-r.ctx.Done():
		r.releaseActiveBatch(req.LeadIDs)
		return models.PipelineRun{}, fmt.Errorf("runner shutting down")
	default:
		r.releaseActiveBatch(req.LeadIDs)
		return models.PipelineRun{}, fmt.Errorf("job queue full")
	}
}

// ActiveRun returns the queued/running run ID for a lead, or "" if none.
func (r *Runner) ActiveRun(leadID string) string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.activeRuns[leadID]
}

var (
	// ErrRunInProgress is returned when a lead already has an active run.
	ErrRunInProgress = fmt.Errorf("lead already has an active run")
)

func (r *Runner) reserveActive(leadID string) error {
	if r.activeRuns[leadID] != "" {
		return ErrRunInProgress
	}
	r.activeRuns[leadID] = "reserved"
	return nil
}

func (r *Runner) releaseActive(leadID string) {
	delete(r.activeRuns, leadID)
}

func (r *Runner) releaseActiveBatch(leadIDs []string) {
	for _, id := range leadIDs {
		delete(r.activeRuns, id)
	}
}

func (r *Runner) setActiveRun(leadID, runID string) {
	r.activeRuns[leadID] = runID
}

func (r *Runner) worker() {
	defer r.wg.Done()
	for {
		select {
		case <-r.ctx.Done():
			return
		case j, ok := <-r.jobs:
			if !ok {
				return
			}
			r.processJob(r.ctx, j)
		}
	}
}

func (r *Runner) processJob(ctx context.Context, j job) {
	defer func() {
		if rec := recover(); rec != nil {
			r.failRun(ctx, j.runID, fmt.Errorf("worker panic: %v", rec))
			if j.leadID != "" {
				r.releaseActive(j.leadID)
			} else if j.batch != nil {
				r.releaseActiveBatch(j.batch.LeadIDs)
			}
		}
	}()

	run, err := r.store.GetPipelineRun(ctx, j.runID)
	if err != nil {
		r.failRun(ctx, j.runID, err)
		r.releaseForJob(j)
		return
	}

	run.Status = "running"
	if _, err := r.store.UpdatePipelineRun(ctx, run); err != nil {
		r.failRun(ctx, j.runID, err)
		r.releaseForJob(j)
		return
	}

	r.markRunning(j)

	if j.single != nil {
		lead, err := r.store.GetLead(ctx, j.leadID)
		if err != nil {
			r.failRun(ctx, j.runID, err)
			r.releaseForJob(j)
			return
		}
		_, _ = r.executeSingle(ctx, lead, run, *j.single)
	} else if j.batch != nil {
		_, _ = r.executeBatch(ctx, run, *j.batch)
	}

	r.releaseForJob(j)
}

func (r *Runner) markRunning(j job) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if j.leadID != "" {
		r.activeRuns[j.leadID] = j.runID
	} else if j.batch != nil {
		for _, id := range j.batch.LeadIDs {
			r.activeRuns[id] = j.runID
		}
	}
}

func (r *Runner) releaseForJob(j job) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if j.leadID != "" {
		delete(r.activeRuns, j.leadID)
	} else if j.batch != nil {
		for _, id := range j.batch.LeadIDs {
			delete(r.activeRuns, id)
		}
	}
}

func (r *Runner) failRun(ctx context.Context, runID string, runErr error) {
	run, err := r.store.GetPipelineRun(ctx, runID)
	if err != nil {
		return
	}
	now := time.Now().UTC()
	run.FinishedAt = &now
	run.Status = "failed"
	run.Error = runErr.Error()
	_, _ = r.store.UpdatePipelineRun(ctx, run)
}
