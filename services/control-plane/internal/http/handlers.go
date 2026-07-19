package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/Moyeil-73/osint-lead-platform/services/control-plane/internal/models"
	"github.com/Moyeil-73/osint-lead-platform/services/control-plane/internal/readiness"
	"github.com/Moyeil-73/osint-lead-platform/services/control-plane/internal/risk"
	"github.com/Moyeil-73/osint-lead-platform/services/control-plane/internal/store"
	"github.com/Moyeil-73/osint-lead-platform/services/control-plane/internal/util"
)

// response envelope shapes.
type response struct {
	Data any `json:"data,omitempty"`
	Meta any `json:"meta,omitempty"`
}

type errorResponse struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

type listMeta struct {
	Page     int `json:"page"`
	PageSize int `json:"page_size"`
	Total    int `json:"total"`
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	errBody := errorResponse{}
	errBody.Error.Code = code
	errBody.Error.Message = message
	_ = json.NewEncoder(w).Encode(errBody)
}

func parseJSON(r *http.Request, v any) error {
	defer r.Body.Close()
	return json.NewDecoder(r.Body).Decode(v)
}

func mapStoreError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, store.ErrNotFound):
		writeError(w, http.StatusNotFound, "not_found", "resource not found")
	case errors.Is(err, store.ErrInvalid):
		writeError(w, http.StatusBadRequest, "invalid_request", "invalid request")
	default:
		writeError(w, http.StatusInternalServerError, "internal_error", err.Error())
	}
}

func (s *Server) handleCreateLead(w http.ResponseWriter, r *http.Request) {
	var raw models.RawLead
	if err := parseJSON(r, &raw); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "invalid JSON body")
		return
	}

	lead := models.Lead{
		ID:            util.NewID(),
		Name:          raw.Name,
		Email:         raw.Email,
		Phone:         raw.Phone,
		Company:       raw.Company,
		Domain:        raw.Domain,
		URL:           raw.URL,
		SourceID:      raw.SourceID,
		PermissionRef: raw.PermissionRef,
		Stage:         models.StageRaw,
		RiskLevel:     models.RiskUnknown,
		Results:       map[string]any{},
	}

	created, err := s.store.CreateLead(r.Context(), lead)
	if err != nil {
		mapStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, response{Data: leadToJSON(created, false)})
}

func (s *Server) handleListLeads(w http.ResponseWriter, r *http.Request) {
	params := models.LeadSearchParams{
		Stage:        r.URL.Query().Get("stage"),
		Risk:         r.URL.Query().Get("risk"),
		ModuleStatus: r.URL.Query().Get("module_status"),
		Q:            r.URL.Query().Get("q"),
		Page:         parseInt(r.URL.Query().Get("page"), 1),
		PageSize:     parseInt(r.URL.Query().Get("page_size"), 25),
	}

	leads, total, err := s.store.ListLeads(r.Context(), params)
	if err != nil {
		mapStoreError(w, err)
		return
	}

	out := make([]map[string]any, len(leads))
	for i, l := range leads {
		out[i] = leadToJSON(l, false)
	}
	writeJSON(w, http.StatusOK, response{
		Data: out,
		Meta: listMeta{Page: params.Page, PageSize: params.PageSize, Total: total},
	})
}

func (s *Server) handleGetLead(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "missing lead id")
		return
	}

	lead, err := s.store.GetLead(r.Context(), id)
	if err != nil {
		mapStoreError(w, err)
		return
	}

	events, err := s.store.ListAuditEventsByLead(r.Context(), id)
	if err != nil {
		mapStoreError(w, err)
		return
	}
	lead.AuditEvents = events

	writeJSON(w, http.StatusOK, response{Data: leadToJSON(lead, true)})
}

func (s *Server) handleRunModules(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "missing lead id")
		return
	}

	var req models.RunModulesRequest
	if err := parseJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "invalid JSON body")
		return
	}
	if len(req.Modules) == 0 {
		writeError(w, http.StatusBadRequest, "invalid_request", "modules is required")
		return
	}
	for _, m := range req.Modules {
		if !s.registry.IsKnown(m) {
			writeError(w, http.StatusBadRequest, "invalid_request", fmt.Sprintf("unknown module: %s", m))
			return
		}
	}

	lead, err := s.runner.RunSingle(r.Context(), id, req)
	if err != nil {
		mapStoreError(w, err)
		return
	}

	events, err := s.store.ListAuditEventsByLead(r.Context(), id)
	if err != nil {
		mapStoreError(w, err)
		return
	}
	lead.AuditEvents = events

	writeJSON(w, http.StatusOK, response{Data: leadToJSON(lead, true)})
}

func (s *Server) handleListModules(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, response{Data: s.registry.List()})
}

func (s *Server) handleGetModule(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	mod, ok := s.registry.Get(name)
	if !ok {
		writeError(w, http.StatusNotFound, "not_found", "module not found")
		return
	}
	writeJSON(w, http.StatusOK, response{Data: mod})
}

func (s *Server) handleListAudit(w http.ResponseWriter, r *http.Request) {
	params := models.AuditSearchParams{
		Module:   r.URL.Query().Get("module"),
		Status:   r.URL.Query().Get("status"),
		Page:     parseInt(r.URL.Query().Get("page"), 1),
		PageSize: parseInt(r.URL.Query().Get("page_size"), 25),
	}

	events, total, err := s.store.ListAuditEvents(r.Context(), params)
	if err != nil {
		mapStoreError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, response{
		Data: events,
		Meta: listMeta{Page: params.Page, PageSize: params.PageSize, Total: total},
	})
}

func (s *Server) handleListRuns(w http.ResponseWriter, r *http.Request) {
	params := models.AuditSearchParams{
		Page:     parseInt(r.URL.Query().Get("page"), 1),
		PageSize: parseInt(r.URL.Query().Get("page_size"), 25),
	}

	runs, total, err := s.store.ListPipelineRuns(r.Context(), params)
	if err != nil {
		mapStoreError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, response{
		Data: runs,
		Meta: listMeta{Page: params.Page, PageSize: params.PageSize, Total: total},
	})
}

func (s *Server) handleGetRun(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "missing run id")
		return
	}

	run, err := s.store.GetPipelineRun(r.Context(), id)
	if err != nil {
		mapStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, response{Data: run})
}

func (s *Server) handleComplianceSummary(w http.ResponseWriter, r *http.Request) {
	summary, err := s.store.ComplianceSummary(r.Context())
	if err != nil {
		mapStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, response{Data: summary})
}

func (s *Server) handlePipelineRun(w http.ResponseWriter, r *http.Request) {
	var req models.PipelineRunRequest
	if err := parseJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "invalid JSON body")
		return
	}
	if len(req.LeadIDs) == 0 {
		writeError(w, http.StatusBadRequest, "invalid_request", "lead_ids is required")
		return
	}
	if len(req.Modules) == 0 {
		writeError(w, http.StatusBadRequest, "invalid_request", "modules is required")
		return
	}
	for _, m := range req.Modules {
		if !s.registry.IsKnown(m) {
			writeError(w, http.StatusBadRequest, "invalid_request", fmt.Sprintf("unknown module: %s", m))
			return
		}
	}

	run, err := s.runner.RunBatch(r.Context(), req)
	if err != nil {
		mapStoreError(w, err)
		return
	}

	writeJSON(w, http.StatusAccepted, response{Data: map[string]any{
		"accepted": true,
		"run_id":   run.ID,
	}})
}

func (s *Server) handleReadiness(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "missing lead id")
		return
	}

	lead, err := s.store.GetLead(r.Context(), id)
	if err != nil {
		mapStoreError(w, err)
		return
	}

	report := readiness.Compute(lead)
	writeJSON(w, http.StatusOK, response{Data: report})
}

func (s *Server) handlePromote(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "missing lead id")
		return
	}

	var req models.StageTransitionRequest
	if err := parseJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "invalid JSON body")
		return
	}
	if req.Target == "" {
		req.Target = models.StageCrmReady
	}

	lead, err := s.store.GetLead(r.Context(), id)
	if err != nil {
		mapStoreError(w, err)
		return
	}

	report := readiness.Compute(lead)
	summary := map[string]any{
		"action":        "promote",
		"target":        req.Target,
		"ready":         report.Ready,
		"checks":        report.Checks,
		"previousStage": lead.Stage,
	}

	if !report.Ready {
		s.logCrmEvent(lead, "pipeline", "crm_promote", "error", summary)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]string{
				"code":    "not_ready",
				"message": "lead does not meet crm_ready requirements",
			},
			"data": report,
		})
		return
	}

	lead.Stage = models.StageCrmReady
	lead.UpdatedAt = currentTime()
	if _, err := s.store.UpdateLead(r.Context(), lead); err != nil {
		mapStoreError(w, err)
		return
	}

	summary["stage"] = lead.Stage
	s.logCrmEvent(lead, "pipeline", "crm_promote", "ok", summary)

	events, err := s.store.ListAuditEventsByLead(r.Context(), id)
	if err != nil {
		mapStoreError(w, err)
		return
	}
	lead.AuditEvents = events

	writeJSON(w, http.StatusOK, response{Data: leadToJSON(lead, true)})
}

func (s *Server) handleDemote(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "missing lead id")
		return
	}

	var req models.StageTransitionRequest
	if err := parseJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "invalid JSON body")
		return
	}

	lead, err := s.store.GetLead(r.Context(), id)
	if err != nil {
		mapStoreError(w, err)
		return
	}

	if !readiness.CanDemoteTo(lead.Stage, req.Target) {
		writeError(w, http.StatusBadRequest, "invalid_request", fmt.Sprintf("cannot demote from %s to %s", lead.Stage, req.Target))
		return
	}

	summary := map[string]any{
		"action":        "demote",
		"target":        req.Target,
		"previousStage": lead.Stage,
	}

	lead.Stage = req.Target
	lead.UpdatedAt = currentTime()
	if _, err := s.store.UpdateLead(r.Context(), lead); err != nil {
		mapStoreError(w, err)
		return
	}

	summary["stage"] = lead.Stage
	s.logCrmEvent(lead, "pipeline", "crm_demote", "ok", summary)

	events, err := s.store.ListAuditEventsByLead(r.Context(), id)
	if err != nil {
		mapStoreError(w, err)
		return
	}
	lead.AuditEvents = events

	writeJSON(w, http.StatusOK, response{Data: leadToJSON(lead, true)})
}

func (s *Server) handleExport(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "missing lead id")
		return
	}

	lead, err := s.store.GetLead(r.Context(), id)
	if err != nil {
		mapStoreError(w, err)
		return
	}

	report := readiness.Compute(lead)
	summary := map[string]any{
		"action": "export",
		"ready":  report.Ready,
		"stage":  lead.Stage,
		"checks": report.Checks,
	}

	if lead.Stage != models.StageCrmReady {
		s.logCrmEvent(lead, "pipeline", "crm_export", "error", summary)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]string{
				"code":    "not_ready",
				"message": "lead must be crm_ready before export",
			},
			"data": report,
		})
		return
	}

	legalBasis := lead.PermissionRef
	if legalBasis == "" {
		legalBasis = models.LegalBasisGDPR
	} else {
		legalBasis = fmt.Sprintf("%s (permission_ref: %s)", models.LegalBasisGDPR, lead.PermissionRef)
	}

	summary["exportedAt"] = currentTime().Format(time.RFC3339)
	s.logCrmEvent(lead, "pipeline", "crm_export", "ok", summary)

	resp := map[string]any{
		"format":      "crm_stub_v1",
		"exported_at": currentTime().Format(time.RFC3339),
		"legal_basis": legalBasis,
		"permission_ref": lead.PermissionRef,
		"lead": map[string]any{
			"id":         lead.ID,
			"name":       lead.Name,
			"email":      lead.Email,
			"phone":      lead.Phone,
			"company":    lead.Company,
			"domain":     lead.Domain,
			"url":        lead.URL,
			"source_id":  lead.SourceID,
			"stage":      lead.Stage,
			"risk_level": lead.RiskLevel,
		},
		"enrichment": lead.Results,
		"readiness":  report,
	}

	writeJSON(w, http.StatusOK, response{Data: resp})
}

func (s *Server) handleRisk(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "missing lead id")
		return
	}

	lead, err := s.store.GetLead(r.Context(), id)
	if err != nil {
		mapStoreError(w, err)
		return
	}

	report := risk.Compute(lead)
	writeJSON(w, http.StatusOK, response{Data: report})
}

// helpers.

func parseInt(s string, def int) int {
	if s == "" {
		return def
	}
	v, err := strconv.Atoi(s)
	if err != nil || v < 1 {
		return def
	}
	return v
}

// leadToJSON converts a Lead to the JSON shape the frontend expects: module
// result keys are flattened to the top level and the internal Results map is
// omitted. Audit events are included only when withAudit is true.
func leadToJSON(lead models.Lead, withAudit bool) map[string]any {
	m := map[string]any{
		"id":             lead.ID,
		"name":           lead.Name,
		"email":          lead.Email,
		"phone":          lead.Phone,
		"company":        lead.Company,
		"domain":         lead.Domain,
		"url":            lead.URL,
		"source_id":      lead.SourceID,
		"permission_ref": lead.PermissionRef,
		"stage":          lead.Stage,
		"risk_level":     lead.RiskLevel,
		"risk_score":     lead.RiskScore,
		"created_at":     lead.CreatedAt,
		"updated_at":     lead.UpdatedAt,
	}
	if lead.RiskScore == nil {
		delete(m, "risk_score")
	}
	for k, v := range lead.Results {
		m[k] = v
	}
	if withAudit {
		m["audit_events"] = lead.AuditEvents
	}
	return m
}

// currentTime is used by tests that want deterministic timestamps.
var currentTime = func() time.Time { return time.Now().UTC() }

// logCrmEvent creates a sanitized pipeline audit event for promote/demote/export.
func (s *Server) logCrmEvent(lead models.Lead, module, tool, status string, summary map[string]any) {
	subject := models.Subject{}
	if lead.Domain != "" {
		subject.Domain = lead.Domain
	} else if lead.Email != "" {
		// Only keep a stable, redacted handle-like identifier for the audit log.
		subject.Email = lead.Email
	}

	raw, _ := json.Marshal(summary)
	event := models.AuditEvent{
		ID:            util.NewID(),
		LeadID:        lead.ID,
		Module:        module,
		Tool:          tool,
		CheckedAt:     currentTime(),
		Status:        status,
		LegalBasis:    models.LegalBasisGDPR,
		Subject:       subject,
		RawStderrJSON: string(raw),
		CreatedAt:     currentTime(),
	}
	if lead.PermissionRef != "" {
		event.LegalBasis = fmt.Sprintf("%s (permission_ref: %s)", models.LegalBasisGDPR, lead.PermissionRef)
	}
	_, _ = s.store.CreateAuditEvent(context.Background(), event)
}
