// Package runner executes module libraries and persists the resulting lead
// updates and audit events. It is the bridge between the HTTP API and the
// domain modules in modules/.
package runner

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	domainintel "github.com/Moyeil-73/osint-lead-platform/modules/domain-intel"
	emailvalidate "github.com/Moyeil-73/osint-lead-platform/modules/email-validate"
	phonevalidate "github.com/Moyeil-73/osint-lead-platform/modules/phone-validate"

	"github.com/Moyeil-73/osint-lead-platform/services/control-plane/internal/models"
	"github.com/Moyeil-73/osint-lead-platform/services/control-plane/internal/store"
	"github.com/Moyeil-73/osint-lead-platform/services/control-plane/internal/util"
)

// socialMinInterval is the minimum time between social-footprint attempts per
// lead. The real social module has its own limiter; this guards the stub path.
const socialMinInterval = 5 * time.Second

// Runner executes modules and persists their output.
type Runner struct {
	store      store.Store
	emailVal   *emailvalidate.Validator
	phoneVal   *phonevalidate.Validator
	domainVal  *domainintel.Analyzer
	socialMu   sync.Mutex
	socialLast map[string]time.Time
}

// New builds a Runner backed by the supplied Store. A non-positive timeout uses
// each module's default.
func New(s store.Store, timeout time.Duration) *Runner {
	return &Runner{
		store:      s,
		emailVal:   emailvalidate.NewValidator(timeout),
		phoneVal:   phonevalidate.NewValidator(timeout),
		domainVal:  domainintel.NewAnalyzer(timeout),
		socialLast: make(map[string]time.Time),
	}
}

// RunSingle runs the requested modules on a single lead and returns the updated
// lead. It creates and finalises a PipelineRun of type "single".
func (r *Runner) RunSingle(ctx context.Context, leadID string, req models.RunModulesRequest) (models.Lead, error) {
	lead, err := r.store.GetLead(ctx, leadID)
	if err != nil {
		return models.Lead{}, err
	}

	legalBasis := req.LegalBasis
	if legalBasis == "" {
		legalBasis = models.LegalBasisGDPR
	}
	permissionRef := req.PermissionRef
	if permissionRef == "" {
		permissionRef = lead.PermissionRef
	}

	run := models.PipelineRun{
		ID:              util.NewID(),
		Type:            "single",
		Status:          "running",
		StartedAt:       time.Now().UTC(),
		LeadIDs:         []string{leadID},
		ModulesExecuted: req.Modules,
		LegalBasis:      legalBasis,
		PermissionRefs:  uniqueStrings(append([]string{}, permissionRef)),
	}
	if _, err := r.store.CreatePipelineRun(ctx, run); err != nil {
		return models.Lead{}, fmt.Errorf("create run: %w", err)
	}

	lead, auditEvents, err := r.runModulesOnLead(ctx, lead, req.Modules, run.ID, legalBasis, permissionRef)
	if err != nil {
		r.finaliseRun(ctx, run, nil, err)
		return models.Lead{}, err
	}

	if _, err := r.store.UpdateLead(ctx, lead); err != nil {
		r.finaliseRun(ctx, run, auditEvents, err)
		return models.Lead{}, fmt.Errorf("update lead: %w", err)
	}

	auditIDs := make([]string, 0, len(auditEvents))
	for _, e := range auditEvents {
		saved, err := r.store.CreateAuditEvent(ctx, e)
		if err != nil {
			r.finaliseRun(ctx, run, auditEvents, err)
			return models.Lead{}, fmt.Errorf("save audit event: %w", err)
		}
		auditIDs = append(auditIDs, saved.ID)
	}

	r.finaliseRun(ctx, run, auditEvents, nil)
	_ = auditIDs
	return lead, nil
}

// RunBatch runs modules across multiple leads and returns the created PipelineRun.
// Rate limiting is applied to the social-footprint module path.
func (r *Runner) RunBatch(ctx context.Context, req models.PipelineRunRequest) (models.PipelineRun, error) {
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

	run := models.PipelineRun{
		ID:              util.NewID(),
		Type:            "batch",
		Status:          "running",
		StartedAt:       time.Now().UTC(),
		LeadIDs:         req.LeadIDs,
		ModulesExecuted: req.Modules,
		LegalBasis:      legalBasis,
		PermissionRefs:  uniqueStrings(append([]string{}, req.PermissionRef)),
	}
	if _, err := r.store.CreatePipelineRun(ctx, run); err != nil {
		return models.PipelineRun{}, fmt.Errorf("create run: %w", err)
	}

	allAuditEvents := []models.AuditEvent{}
	hasError := false
	for _, leadID := range req.LeadIDs {
		lead, err := r.store.GetLead(ctx, leadID)
		if err != nil {
			hasError = true
			continue
		}

		permissionRef := req.PermissionRef
		if permissionRef == "" {
			permissionRef = lead.PermissionRef
		}

		updated, events, err := r.runModulesOnLead(ctx, lead, req.Modules, run.ID, legalBasis, permissionRef)
		if err != nil {
			hasError = true
			continue
		}

		if _, err := r.store.UpdateLead(ctx, updated); err != nil {
			hasError = true
			continue
		}

		for _, e := range events {
			saved, err := r.store.CreateAuditEvent(ctx, e)
			if err != nil {
				hasError = true
				continue
			}
			run.AuditEventIDs = append(run.AuditEventIDs, saved.ID)
			allAuditEvents = append(allAuditEvents, saved)
		}
	}

	if hasError {
		run.Status = "partial"
		run.Error = "one or more leads could not be processed"
	} else {
		run.Status = "completed"
	}
	now := time.Now().UTC()
	run.FinishedAt = &now

	updated, err := r.store.UpdatePipelineRun(ctx, run)
	if err != nil {
		return run, fmt.Errorf("update run: %w", err)
	}
	return updated, nil
}

// runModulesOnLead executes the requested modules, updates the lead's results and
// stage/risk, and returns the lead plus audit events.
func (r *Runner) runModulesOnLead(ctx context.Context, lead models.Lead, modules []string, runID, legalBasis, permissionRef string) (models.Lead, []models.AuditEvent, error) {
	if lead.Results == nil {
		lead.Results = map[string]any{}
	}

	var auditEvents []models.AuditEvent
	executed := make([]string, 0, len(modules))
	moduleStatuses := map[string]string{}

	for _, name := range modules {
		name = strings.ToLower(strings.TrimSpace(name))
		if name == "" {
			continue
		}
		executed = append(executed, name)

		res, err := r.runModule(ctx, lead, name, runID, legalBasis)
		if err != nil {
			return lead, auditEvents, err
		}
		if res.Result != nil {
			m, err := resultToMap(res.Result)
			if err != nil {
				return lead, auditEvents, fmt.Errorf("convert %s result: %w", name, err)
			}
			res.Result = m
			lead.Results[res.Key] = m
		}
		auditEvents = append(auditEvents, res.AuditEvents...)

		status := moduleResultStatus(res.Result)
		if status != "" {
			moduleStatuses[name] = status
		}
	}

	lead.Stage = computeStage(lead.Stage, executed, moduleStatuses)
	lead.RiskLevel = computeRisk(lead.Results)
	_ = moduleStatuses
	_ = permissionRef
	_ = ctx
	return lead, auditEvents, nil
}

func (r *Runner) runModule(ctx context.Context, lead models.Lead, name, runID, legalBasis string) (models.ModuleResult, error) {
	checkedAt := time.Now().UTC()

	switch name {
	case models.ModuleEmailValidate:
		res, audit := r.emailVal.Validate(lead.Email)
		raw, _ := json.Marshal(audit)
		checkedTime, _ := time.Parse(time.RFC3339, audit.CheckedAt)
		if checkedTime.IsZero() {
			checkedTime = checkedAt
		}
		event := models.AuditEvent{
			ID:            util.NewID(),
			LeadID:        lead.ID,
			RunID:         &runID,
			Module:        models.ModuleEmailValidate,
			Tool:          audit.Tool,
			CheckedAt:     checkedTime,
			Status:        audit.Status,
			LegalBasis:    audit.LegalBasis,
			Subject:       models.Subject{Email: audit.Email},
			RawStderrJSON: string(raw),
		}
		return models.ModuleResult{Key: "email_validate", Result: res, AuditEvents: []models.AuditEvent{event}}, nil

	case models.ModulePhoneValidate:
		res, audits := r.phoneVal.Validate(lead.Phone)
		events := make([]models.AuditEvent, 0, len(audits))
		for _, a := range audits {
			auditRaw, _ := json.Marshal(a)
			checkedTime, _ := time.Parse(time.RFC3339, a.CheckedAt)
			if checkedTime.IsZero() {
				checkedTime = checkedAt
			}
			events = append(events, models.AuditEvent{
				ID:            util.NewID(),
				LeadID:        lead.ID,
				RunID:         &runID,
				Module:        models.ModulePhoneValidate,
				Tool:          a.Tool,
				CheckedAt:     checkedTime,
				Status:        a.Status,
				LegalBasis:    a.LegalBasis,
				Subject:       models.Subject{PhoneRedacted: a.Phone},
				RawStderrJSON: string(auditRaw),
			})
		}
		return models.ModuleResult{Key: "phone_validate", Result: res, AuditEvents: events}, nil

	case models.ModuleSocialFootprint:
		subject := models.Subject{}
		reason := "not wired in control-plane v1"
		if !r.socialAllowed(lead.ID) {
			reason = "rate limited"
		}
		raw, _ := json.Marshal(map[string]string{"reason": reason})
		event := models.AuditEvent{
			ID:            util.NewID(),
			LeadID:        lead.ID,
			RunID:         &runID,
			Module:        models.ModuleSocialFootprint,
			Tool:          "control-plane",
			CheckedAt:     checkedAt,
			Status:        "skipped",
			LegalBasis:    legalBasis,
			Subject:       subject,
			RawStderrJSON: string(raw),
		}
		result := map[string]any{
			"status":  "skipped",
			"reason":  reason,
			"handles": []any{},
		}
		return models.ModuleResult{Key: "social_footprint", Result: result, AuditEvents: []models.AuditEvent{event}}, nil

	case models.ModuleDomainIntel:
		if lead.Domain == "" {
			raw, _ := json.Marshal(map[string]string{"reason": "missing domain"})
			event := models.AuditEvent{
				ID:            util.NewID(),
				LeadID:        lead.ID,
				RunID:         &runID,
				Module:        models.ModuleDomainIntel,
				Tool:          "control-plane",
				CheckedAt:     checkedAt,
				Status:        "skipped",
				LegalBasis:    legalBasis,
				Subject:       models.Subject{Domain: lead.Domain},
				RawStderrJSON: string(raw),
			}
			return models.ModuleResult{
				Key:         "domain_intel",
				Result:      map[string]any{"status": "skipped", "reason": "missing domain"},
				AuditEvents: []models.AuditEvent{event},
			}, nil
		}

		res, audits := r.domainVal.Analyze(lead.Domain)
		m, err := resultToMap(res)
		if err != nil {
			return models.ModuleResult{}, fmt.Errorf("convert domain-intel result: %w", err)
		}

		status := "unknown"
		if res.WebCheck.Status == "ok" || res.Harvester.Status == "ok" {
			status = "ok"
		}
		m["status"] = status

		events := make([]models.AuditEvent, 0, len(audits))
		for _, a := range audits {
			auditRaw, _ := json.Marshal(a)
			auditTime, _ := time.Parse(time.RFC3339, a.CheckedAt)
			if auditTime.IsZero() {
				auditTime = checkedAt
			}
			events = append(events, models.AuditEvent{
				ID:            util.NewID(),
				LeadID:        lead.ID,
				RunID:         &runID,
				Module:        models.ModuleDomainIntel,
				Tool:          a.Tool,
				CheckedAt:     auditTime,
				Status:        a.Status,
				LegalBasis:    a.LegalBasis,
				Subject:       models.Subject{Domain: a.Domain},
				RawStderrJSON: string(auditRaw),
			})
		}
		return models.ModuleResult{Key: "domain_intel", Result: m, AuditEvents: events}, nil

	default:
		// extraction, company-enrich and any unknown module: stub.
		raw, _ := json.Marshal(map[string]string{"reason": "not wired in control-plane v1"})
		event := models.AuditEvent{
			ID:            util.NewID(),
			LeadID:        lead.ID,
			RunID:         &runID,
			Module:        name,
			Tool:          "control-plane",
			CheckedAt:     checkedAt,
			Status:        "skipped",
			LegalBasis:    legalBasis,
			Subject:       models.Subject{},
			RawStderrJSON: string(raw),
		}
		return models.ModuleResult{Key: strings.ReplaceAll(name, "-", "_"), Result: map[string]any{
			"status": "skipped",
			"reason": "not wired in control-plane v1",
		}, AuditEvents: []models.AuditEvent{event}}, nil
	}
}

func (r *Runner) socialAllowed(leadID string) bool {
	r.socialMu.Lock()
	defer r.socialMu.Unlock()
	last, ok := r.socialLast[leadID]
	allowed := !ok || time.Since(last) >= socialMinInterval
	if allowed {
		r.socialLast[leadID] = time.Now().UTC()
	}
	return allowed
}

func (r *Runner) finaliseRun(ctx context.Context, run models.PipelineRun, events []models.AuditEvent, runErr error) {
	now := time.Now().UTC()
	run.FinishedAt = &now
	if runErr != nil {
		run.Status = "failed"
		run.Error = runErr.Error()
	} else {
		run.Status = "completed"
	}
	for _, e := range events {
		run.AuditEventIDs = append(run.AuditEventIDs, e.ID)
	}
	_, _ = r.store.UpdatePipelineRun(ctx, run)
}

func computeStage(current string, executed []string, statuses map[string]string) string {
	stageOrder := map[string]int{
		models.StageRaw:       0,
		models.StageEnriched:  1,
		models.StageValidated: 2,
		models.StageCrmReady:  3,
	}
	currentOrder, ok := stageOrder[current]
	if !ok {
		currentOrder = -1
	}

	for _, name := range executed {
		if statuses[name] != "ok" {
			// Do not advance stage on skipped/unknown modules.
			continue
		}
		order := -1
		switch name {
		case models.ModuleDomainIntel, models.ModuleExtraction, models.ModuleCompanyEnrich:
			order = stageOrder[models.StageEnriched]
		case models.ModuleEmailValidate, models.ModulePhoneValidate, models.ModuleSocialFootprint:
			order = stageOrder[models.StageValidated]
		}
		if order > currentOrder {
			currentOrder = order
		}
	}

	for stage, order := range stageOrder {
		if order == currentOrder {
			return stage
		}
	}
	return models.StageRaw
}

func computeRisk(results map[string]any) string {
	priority := map[string]int{models.RiskUnknown: 0, models.RiskLow: 1, models.RiskMedium: 2, models.RiskHigh: 3}
	max := models.RiskUnknown
	maxP := priority[max]

	if email, ok := results["email_validate"].(map[string]any); ok {
		if r := emailRisk(email); priority[r] > maxP {
			max = r
			maxP = priority[r]
		}
	}
	if phone, ok := results["phone_validate"].(map[string]any); ok {
		if r := phoneRisk(phone); priority[r] > maxP {
			max = r
			maxP = priority[r]
		}
	}

	return max
}

func emailRisk(email map[string]any) string {
	status, _ := email["status"].(string)
	if status != "ok" {
		return models.RiskMedium
	}
	syntax, _ := email["syntax_valid"].(bool)
	hasMX, _ := email["has_mx_records"].(bool)
	if syntax && hasMX {
		return models.RiskLow
	}
	if syntax {
		return models.RiskMedium
	}
	return models.RiskHigh
}

func phoneRisk(phone map[string]any) string {
	status, _ := phone["status"].(string)
	if status != "ok" {
		return models.RiskMedium
	}
	valid, _ := phone["is_valid_number"].(bool)
	if valid {
		return models.RiskLow
	}
	format, _ := phone["format_valid"].(bool)
	if format {
		return models.RiskMedium
	}
	return models.RiskHigh
}

func moduleResultStatus(result any) string {
	m, ok := result.(map[string]any)
	if !ok {
		return ""
	}
	s, _ := m["status"].(string)
	return s
}

func resultToMap(v any) (map[string]any, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, err
	}
	if m == nil {
		m = map[string]any{}
	}
	return m, nil
}

func uniqueStrings(in []string) []string {
	seen := make(map[string]struct{})
	out := make([]string, 0, len(in))
	for _, s := range in {
		if s == "" {
			continue
		}
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	return out
}
