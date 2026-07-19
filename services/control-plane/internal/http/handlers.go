package httpapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/Moyeil-73/osint-lead-platform/services/control-plane/internal/models"
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
