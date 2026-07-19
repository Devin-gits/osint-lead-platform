package extraction

import (
	"context"
	"net"
	"strings"
	"testing"
	"time"
)

// fakePublicResolve always returns a public IPv4 address. It is used in unit
// tests so they remain network-independent while still exercising DNS-level
// IP validation.
func fakePublicResolve(host string) ([]net.IP, error) {
	return []net.IP{net.ParseIP("93.184.216.34")}, nil
}

// fakePrivateResolve simulates a hostname that resolves to a private IP.
func fakePrivateResolve(host string) ([]net.IP, error) {
	return []net.IP{net.ParseIP("10.0.0.1")}, nil
}

type fakeRunner struct {
	status string
	err    string
	fields Fields
}

func (f *fakeRunner) run(ctx context.Context, url string, timeout time.Duration) (Result, error) {
	return Result{
		Status:     f.status,
		URL:        url,
		FinalURL:   url,
		SourceTool: "fake-runner",
		Fields:     f.fields,
		Metadata:   Metadata{Backend: "fake"},
		Error:      f.err,
	}, nil
}

func (f *fakeRunner) sourceTool() string { return "fake-runner" }

func newTestExtractor(runner runner) *Extractor {
	return &Extractor{
		backend: BackendCrawl4AI,
		runner:  runner,
		timeout: time.Second,
		limiter: newRateLimiter(0),
		resolve: fakePublicResolve,
	}
}

func TestNewExtractor_DefaultBackendIsCrawl4AI(t *testing.T) {
	e := NewExtractor(0, 0, "")
	if e.backend != BackendCrawl4AI {
		t.Errorf("backend = %q, want %q", e.backend, BackendCrawl4AI)
	}
}

func TestNewExtractor_EnvBackendOverrides(t *testing.T) {
	t.Setenv("EXTRACTION_BACKEND", "firecrawl")
	e := NewExtractor(0, 0, "")
	if e.backend != BackendFirecrawl {
		t.Errorf("backend = %q, want firecrawl", e.backend)
	}
}

func TestExtractor_MissingPermissionRef(t *testing.T) {
	e := newTestExtractor(&fakeRunner{})
	res, audit := e.Extract(context.Background(), Input{URL: "https://example.com"})
	if res.Status != "skipped" {
		t.Errorf("status = %q, want skipped", res.Status)
	}
	if res.Error != "no permission_ref provided" {
		t.Errorf("error = %q, want no permission_ref provided", res.Error)
	}
	if audit.PermissionRef != "" {
		t.Errorf("audit permission_ref = %q, want empty", audit.PermissionRef)
	}
	if audit.Status != "skipped" {
		t.Errorf("audit status = %q, want skipped", audit.Status)
	}
}

func TestExtractor_MissingURL(t *testing.T) {
	e := newTestExtractor(&fakeRunner{})
	res, audit := e.Extract(context.Background(), Input{PermissionRef: "T-1"})
	if res.Status != "skipped" {
		t.Errorf("status = %q, want skipped", res.Status)
	}
	if audit.RequestURL != "" {
		t.Errorf("audit request_url = %q, want empty", audit.RequestURL)
	}
	if audit.Status != "skipped" {
		t.Errorf("audit status = %q, want skipped", audit.Status)
	}
	if audit.PermissionRef != "T-1" {
		t.Errorf("audit permission_ref = %q, want T-1", audit.PermissionRef)
	}
}

func TestExtractor_BackendSwitchUsesRunner(t *testing.T) {
	fr := &fakeRunner{status: "ok", fields: Fields{CompanyName: "Acme"}}
	e := newTestExtractor(fr)
	res, audit := e.Extract(context.Background(), Input{URL: "https://example.com", PermissionRef: "T-1"})
	if res.Fields.CompanyName != "Acme" {
		t.Errorf("fields not populated: %+v", res.Fields)
	}
	if audit.RequestURL != "https://example.com" {
		t.Errorf("audit request_url = %q", audit.RequestURL)
	}
	if audit.LegalBasis != LegalBasis {
		t.Errorf("audit legal basis = %q", audit.LegalBasis)
	}
	if audit.PermissionRef != "T-1" {
		t.Errorf("audit permission_ref = %q", audit.PermissionRef)
	}
	if audit.Module != "extraction" {
		t.Errorf("audit module = %q", audit.Module)
	}
	if audit.Tool != "fake-runner" {
		t.Errorf("audit tool = %q", audit.Tool)
	}
	if audit.DurationMs < 0 {
		t.Errorf("audit duration_ms = %d", audit.DurationMs)
	}
	if audit.Limits != LimitsApplied {
		t.Errorf("audit limits = %q", audit.Limits)
	}
}

func TestExtractor_RejectedCredentialedURL(t *testing.T) {
	e := newTestExtractor(&fakeRunner{})
	res, audit := e.Extract(context.Background(), Input{URL: "https://user:pass@example.com/", PermissionRef: "T-1"})
	if res.Status != "skipped" {
		t.Errorf("status = %q, want skipped", res.Status)
	}
	if !strings.Contains(res.Error, "credentials") {
		t.Errorf("error = %q, want credentials rejection", res.Error)
	}
	if audit.Status != "skipped" {
		t.Errorf("audit status = %q", audit.Status)
	}
}

func TestExtractor_RejectedPrivateIP(t *testing.T) {
	e := newTestExtractor(&fakeRunner{})
	e.resolve = fakePrivateResolve
	res, _ := e.Extract(context.Background(), Input{URL: "https://example.com/", PermissionRef: "T-1"})
	if res.Status != "skipped" {
		t.Errorf("status = %q, want skipped", res.Status)
	}
	if !strings.Contains(res.Error, "forbidden IP") {
		t.Errorf("error = %q, want forbidden IP", res.Error)
	}
}

func TestExtractor_RejectedIPLiteral(t *testing.T) {
	e := newTestExtractor(&fakeRunner{})
	res, _ := e.Extract(context.Background(), Input{URL: "https://127.0.0.1/", PermissionRef: "T-1"})
	if res.Status != "skipped" {
		t.Errorf("status = %q, want skipped", res.Status)
	}
	if !strings.Contains(res.Error, "IP-literal") {
		t.Errorf("error = %q, want IP-literal rejection", res.Error)
	}
}

func TestExtractor_RejectedMetadataEndpoint(t *testing.T) {
	e := newTestExtractor(&fakeRunner{})
	res, _ := e.Extract(context.Background(), Input{URL: "https://169.254.169.254/", PermissionRef: "T-1"})
	if res.Status != "skipped" {
		t.Errorf("status = %q, want skipped", res.Status)
	}
	if !strings.Contains(res.Error, "forbidden IP") {
		t.Errorf("error = %q, want forbidden IP", res.Error)
	}
}

func TestExtractor_RejectedNonStandardPort(t *testing.T) {
	e := newTestExtractor(&fakeRunner{})
	res, _ := e.Extract(context.Background(), Input{URL: "https://example.com:8080/", PermissionRef: "T-1"})
	if res.Status != "skipped" {
		t.Errorf("status = %q, want skipped", res.Status)
	}
	if !strings.Contains(res.Error, "non-standard port") {
		t.Errorf("error = %q, want non-standard port", res.Error)
	}
}

func TestExtractor_QueryStringRedactedInAudit(t *testing.T) {
	fr := &fakeRunner{status: "ok", fields: Fields{CompanyName: "Acme"}}
	e := newTestExtractor(fr)
	_, audit := e.Extract(context.Background(), Input{URL: "https://example.com/?utm_source=foo&id=123", PermissionRef: "T-1"})
	// The audit query values are redacted; brackets may be URL-encoded by url.Values.Encode.
	if !strings.Contains(audit.RequestURL, "[redacted]") && !strings.Contains(audit.RequestURL, "%5Bredacted%5D") {
		t.Errorf("audit request_url not redacted: %q", audit.RequestURL)
	}
	if strings.Contains(audit.RequestURL, "utm_source=foo") || strings.Contains(audit.RequestURL, "id=123") {
		t.Errorf("audit request_url leaked query values: %q", audit.RequestURL)
	}
}

func TestExtractor_ProvenanceBuilt(t *testing.T) {
	fr := &fakeRunner{status: "ok", fields: Fields{CompanyName: "Acme", Emails: []string{"a@b.com"}}}
	e := newTestExtractor(fr)
	res, _ := e.Extract(context.Background(), Input{URL: "https://example.com/", PermissionRef: "T-1"})
	if len(res.Provenance) < 2 {
		t.Fatalf("provenance too short: %+v", res.Provenance)
	}
	found := map[string]bool{}
	for _, p := range res.Provenance {
		found[p.Field] = true
		if p.SourceURL == "" {
			t.Errorf("provenance source_url empty for field %q", p.Field)
		}
		if p.Timestamp == "" {
			t.Errorf("provenance timestamp empty for field %q", p.Field)
		}
	}
	if !found["company_name"] || !found["emails"] {
		t.Errorf("provenance missing expected fields: %+v", res.Provenance)
	}
}

func TestExtractor_MetadataHasPermissionRefAndLimits(t *testing.T) {
	fr := &fakeRunner{status: "ok", fields: Fields{}}
	e := newTestExtractor(fr)
	res, _ := e.Extract(context.Background(), Input{URL: "https://example.com/", PermissionRef: "T-1"})
	if res.Metadata.PermissionRef != "T-1" {
		t.Errorf("metadata.permission_ref = %q", res.Metadata.PermissionRef)
	}
	if res.Metadata.LimitsApplied != LimitsApplied {
		t.Errorf("metadata.limits_applied = %q", res.Metadata.LimitsApplied)
	}
	if res.Metadata.LegalBasis != LegalBasis {
		t.Errorf("metadata.legal_basis = %q", res.Metadata.LegalBasis)
	}
}

func TestConfidenceFor(t *testing.T) {
	if confidenceFor(Fields{}) != 0.0 {
		t.Errorf("empty fields confidence = %v, want 0", confidenceFor(Fields{}))
	}
	c := confidenceFor(Fields{CompanyName: "Acme", Title: "Home", Emails: []string{"a@b.com"}})
	want := 3.0 / 7.0
	if c != want {
		t.Errorf("confidence = %v, want %v", c, want)
	}
}
