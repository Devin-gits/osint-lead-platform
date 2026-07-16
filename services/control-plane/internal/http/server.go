package httpapi

import (
	"net/http"

	"github.com/Moyeil-73/osint-lead-platform/services/control-plane/internal/registry"
	"github.com/Moyeil-73/osint-lead-platform/services/control-plane/internal/runner"
	"github.com/Moyeil-73/osint-lead-platform/services/control-plane/internal/store"
)

// Server wires the HTTP handlers with their dependencies.
type Server struct {
	store    store.Store
	runner   *runner.Runner
	registry *registry.Registry
}

// NewServer builds a Server. The store must already be initialised.
func NewServer(s store.Store, r *runner.Runner, reg *registry.Registry) *Server {
	return &Server{store: s, runner: r, registry: reg}
}

// Handler returns the root http.Handler with routes and middleware registered.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("POST /api/leads", s.handleCreateLead)
	mux.HandleFunc("GET /api/leads", s.handleListLeads)
	mux.HandleFunc("GET /api/leads/{id}", s.handleGetLead)
	mux.HandleFunc("POST /api/leads/{id}/run", s.handleRunModules)

	mux.HandleFunc("GET /api/modules", s.handleListModules)
	mux.HandleFunc("GET /api/modules/{name}", s.handleGetModule)

	mux.HandleFunc("GET /api/audit", s.handleListAudit)
	mux.HandleFunc("GET /api/runs", s.handleListRuns)
	mux.HandleFunc("GET /api/runs/{id}", s.handleGetRun)
	mux.HandleFunc("GET /api/compliance/summary", s.handleComplianceSummary)
	mux.HandleFunc("POST /api/pipelines/run", s.handlePipelineRun)

	return withCORS(mux)
}

// ListenAndServe starts the HTTP server on addr.
func (s *Server) ListenAndServe(addr string) error {
	return http.ListenAndServe(addr, s.Handler())
}
