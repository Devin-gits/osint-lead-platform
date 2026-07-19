package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Moyeil-73/osint-lead-platform/services/control-plane/internal/models"
	"github.com/Moyeil-73/osint-lead-platform/services/control-plane/internal/registry"
	"github.com/Moyeil-73/osint-lead-platform/services/control-plane/internal/runner"
	"github.com/Moyeil-73/osint-lead-platform/services/control-plane/internal/store"
	"github.com/Moyeil-73/osint-lead-platform/services/control-plane/internal/util"
)

func newTestServer(t *testing.T) *Server {
	t.Helper()
	st := store.NewMemoryStore()
	r := runner.New(st, 0)
	reg := registry.New()
	return NewServer(st, r, reg)
}

func TestServer_CreateAndGetLead(t *testing.T) {
	srv := newTestServer(t)
	h := srv.Handler()

	body := []byte(`{"email":"support@github.com","company":"GitHub"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/leads", bytes.NewReader(body))
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rr.Code, rr.Body.String())
	}

	var createResp response
	if err := json.Unmarshal(rr.Body.Bytes(), &createResp); err != nil {
		t.Fatalf("decode create response: %v", err)
	}
	lead, ok := createResp.Data.(map[string]any)
	if !ok {
		t.Fatalf("expected object data")
	}
	id, _ := lead["id"].(string)
	if id == "" {
		t.Fatalf("expected lead id")
	}

	req = httptest.NewRequest(http.MethodGet, "/api/leads/"+id, nil)
	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestServer_RunModules(t *testing.T) {
	srv := newTestServer(t)
	h := srv.Handler()

	// Create lead.
	body := []byte(`{"email":"support@github.com"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/leads", bytes.NewReader(body))
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	var createResp response
	if err := json.Unmarshal(rr.Body.Bytes(), &createResp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	lead := createResp.Data.(map[string]any)
	id := lead["id"].(string)

	// Run email-validate.
	body = []byte(`{"modules":["email-validate"]}`)
	req = httptest.NewRequest(http.MethodPost, "/api/leads/"+id+"/run", bytes.NewReader(body))
	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var runResp response
	if err := json.Unmarshal(rr.Body.Bytes(), &runResp); err != nil {
		t.Fatalf("decode run response: %v", err)
	}
	updated := runResp.Data.(map[string]any)
	ev := updated["email_validate"].(map[string]any)
	if ev["status"] != "ok" {
		t.Fatalf("expected email status ok, got %v", ev["status"])
	}
}

func TestServer_ListModules(t *testing.T) {
	srv := newTestServer(t)
	h := srv.Handler()

	req := httptest.NewRequest(http.MethodGet, "/api/modules", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp response
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	mods, ok := resp.Data.([]any)
	if !ok || len(mods) == 0 {
		t.Fatalf("expected modules list")
	}

	found := false
	for _, m := range mods {
		mod, ok := m.(map[string]any)
		if !ok {
			continue
		}
		if mod["name"] == "extraction" {
			found = true
			if mod["dev_status"] != "available" {
				t.Fatalf("expected extraction dev_status available, got %v", mod["dev_status"])
			}
			if mod["min_input_field"] != "url" {
				t.Fatalf("expected extraction min_input_field url, got %v", mod["min_input_field"])
			}
		}
	}
	if !found {
		t.Fatalf("extraction module not found in list: %+v", mods)
	}
}

func TestServer_CORS(t *testing.T) {
	srv := newTestServer(t)
	h := srv.Handler()

	req := httptest.NewRequest(http.MethodOptions, "/api/leads", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", rr.Code)
	}
	if got := rr.Header().Get("Access-Control-Allow-Origin"); got != "http://localhost:3000" {
		t.Fatalf("expected CORS origin, got %s", got)
	}
}

func TestServer_PipelineRun(t *testing.T) {
	srv := newTestServer(t)
	h := srv.Handler()

	// Create lead.
	body := []byte(`{"email":"support@github.com"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/leads", bytes.NewReader(body))
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	var createResp response
	if err := json.Unmarshal(rr.Body.Bytes(), &createResp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	lead := createResp.Data.(map[string]any)
	id := lead["id"].(string)

	body = []byte(`{"lead_ids":["` + id + `"],"modules":["email-validate"]}`)
	req = httptest.NewRequest(http.MethodPost, "/api/pipelines/run", bytes.NewReader(body))
	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d: %s", rr.Code, rr.Body.String())
	}
}

// Ensure unused imports are actually used.
var _ = context.Background
var _ = util.NewID
var _ = models.StageRaw

func TestServer_CRMReadyFlow(t *testing.T) {
	srv := newTestServer(t)
	h := srv.Handler()

	// Create lead ready for promotion.
	body := []byte(`{"email":"support@example.com","company":"Example","domain":"example.com","permission_ref":"p-001"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/leads", bytes.NewReader(body))
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rr.Code, rr.Body.String())
	}
	var createResp response
	if err := json.Unmarshal(rr.Body.Bytes(), &createResp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	lead := createResp.Data.(map[string]any)
	id := lead["id"].(string)

	// Without validation, promotion should fail with 409.
	req = httptest.NewRequest(http.MethodPost, "/api/leads/"+id+"/promote", bytes.NewReader([]byte(`{"target":"crm_ready"}`)))
	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusConflict {
		t.Fatalf("expected 409 before validation, got %d: %s", rr.Code, rr.Body.String())
	}

	// Run email-validate.
	body = []byte(`{"modules":["email-validate"]}`)
	req = httptest.NewRequest(http.MethodPost, "/api/leads/"+id+"/run", bytes.NewReader(body))
	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	// Run company-enrich.
	body = []byte(`{"modules":["company-enrich"]}`)
	req = httptest.NewRequest(http.MethodPost, "/api/leads/"+id+"/run", bytes.NewReader(body))
	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	// Readiness should now pass.
	req = httptest.NewRequest(http.MethodGet, "/api/leads/"+id+"/readiness", nil)
	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	var readyResp response
	if err := json.Unmarshal(rr.Body.Bytes(), &readyResp); err != nil {
		t.Fatalf("decode readiness: %v", err)
	}
	report, ok := readyResp.Data.(map[string]any)
	if !ok {
		t.Fatalf("expected readiness report")
	}
	if report["ready"] != true {
		t.Fatalf("expected ready true, got %v", report["ready"])
	}

	// Promote should succeed.
	body = []byte(`{"target":"crm_ready"}`)
	req = httptest.NewRequest(http.MethodPost, "/api/leads/"+id+"/promote", bytes.NewReader(body))
	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	var promoteResp response
	if err := json.Unmarshal(rr.Body.Bytes(), &promoteResp); err != nil {
		t.Fatalf("decode promote: %v", err)
	}
	promoted := promoteResp.Data.(map[string]any)
	if promoted["stage"] != "crm_ready" {
		t.Fatalf("expected stage crm_ready, got %v", promoted["stage"])
	}

	// Export should now succeed.
	req = httptest.NewRequest(http.MethodGet, "/api/leads/"+id+"/export", nil)
	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	var exportResp response
	if err := json.Unmarshal(rr.Body.Bytes(), &exportResp); err != nil {
		t.Fatalf("decode export: %v", err)
	}
	export := exportResp.Data.(map[string]any)
	if export["format"] != "crm_stub_v1" {
		t.Fatalf("expected format crm_stub_v1, got %v", export["format"])
	}

	// Demote to validated.
	body = []byte(`{"target":"validated"}`)
	req = httptest.NewRequest(http.MethodPost, "/api/leads/"+id+"/demote", bytes.NewReader(body))
	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	var demoteResp response
	if err := json.Unmarshal(rr.Body.Bytes(), &demoteResp); err != nil {
		t.Fatalf("decode demote: %v", err)
	}
	demoted := demoteResp.Data.(map[string]any)
	if demoted["stage"] != "validated" {
		t.Fatalf("expected stage validated, got %v", demoted["stage"])
	}

	// Export now 409.
	req = httptest.NewRequest(http.MethodGet, "/api/leads/"+id+"/export", nil)
	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusConflict {
		t.Fatalf("expected 409 after demote, got %d", rr.Code)
	}
}

func TestServer_ExportNotReady(t *testing.T) {
	srv := newTestServer(t)
	h := srv.Handler()

	body := []byte(`{"email":"support@example.com","company":"Example","permission_ref":"p-001"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/leads", bytes.NewReader(body))
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rr.Code, rr.Body.String())
	}
	var createResp response
	if err := json.Unmarshal(rr.Body.Bytes(), &createResp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	id := createResp.Data.(map[string]any)["id"].(string)

	req = httptest.NewRequest(http.MethodGet, "/api/leads/"+id+"/export", nil)
	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d: %s", rr.Code, rr.Body.String())
	}
}
