package extraction

import (
	"context"
	"testing"
	"time"
)

type fakeRunner struct {
	status string
	err    string
	fields Fields
}

func (f *fakeRunner) run(ctx context.Context, url string, timeout time.Duration) (Result, error) {
	return Result{
		Status:     f.status,
		URL:        url,
		SourceTool: "fake-runner",
		Fields:     f.fields,
		Metadata:   Metadata{Backend: "fake"},
		Error:      f.err,
	}, nil
}

func (f *fakeRunner) sourceTool() string { return "fake-runner" }

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

func TestExtractor_MissingURL(t *testing.T) {
	e := NewExtractor(time.Second, 0, "")
	res, audit := e.Extract(context.Background(), Input{})
	if res.Status != "skipped" {
		t.Errorf("status = %q, want skipped", res.Status)
	}
	if audit.URL != "" {
		t.Errorf("audit URL = %q, want empty", audit.URL)
	}
	if audit.Status != "skipped" {
		t.Errorf("audit status = %q, want skipped", audit.Status)
	}
}

func TestExtractor_BackendSwitchUsesRunner(t *testing.T) {
	fr := &fakeRunner{status: "ok", fields: Fields{CompanyName: "Acme"}}
	e := &Extractor{backend: BackendCrawl4AI, runner: fr, timeout: time.Second, limiter: newRateLimiter(0)}
	res, audit := e.Extract(context.Background(), Input{URL: "https://example.com"})
	if res.Fields.CompanyName != "Acme" {
		t.Errorf("fields not populated: %+v", res.Fields)
	}
	if audit.URL != "https://example.com" {
		t.Errorf("audit URL = %q", audit.URL)
	}
	if audit.LegalBasis != LegalBasis {
		t.Errorf("audit legal basis = %q", audit.LegalBasis)
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
