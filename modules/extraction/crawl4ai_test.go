package extraction

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func fakeCrawl4AIWrapper(t *testing.T, script string) string {
	t.Helper()
	dir := t.TempDir()
	p := filepath.Join(dir, "fake_crawl4ai_extract.py")
	if err := os.WriteFile(p, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestCrawl4AIRunner_UsesExplicitWrapper(t *testing.T) {
	script := `import sys, json
print(json.dumps({
    "status": "ok",
    "url": "https://example.com",
    "final_url": "https://example.com/",
    "source_tool": "unclecode/crawl4ai@v0.9.2 (CLI subprocess)",
    "confidence": 0.5,
    "fields": {"company_name":"Acme Inc.","emails":["hello@acme.com"],"phones":[],"addresses":[],"social_links":[],"contact_urls":[],"description":"","title":"Acme"},
    "raw_markdown": "Acme Inc. hello@acme.com",
    "metadata": {"backend":"crawl4ai","http_status":200,"truncated":False,"raw_bytes":26},
    "error": "",
    "checked_at": "2026-07-16T12:00:00Z"
}))
`
	wrapper := fakeCrawl4AIWrapper(t, script)
	t.Setenv("EXTRACTION_CRAWL4AI_WRAPPER", wrapper)
	r := newCrawl4AIRunner()
	res, err := r.run(context.Background(), "https://example.com", 30*time.Second)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Status != "ok" {
		t.Errorf("status = %q, want ok", res.Status)
	}
	if res.Fields.CompanyName != "Acme Inc." {
		t.Errorf("company_name = %q", res.Fields.CompanyName)
	}
}

func TestCrawl4AIRunner_MissingPython(t *testing.T) {
	t.Setenv("EXTRACTION_CRAWL4AI_PYTHON", "does_not_exist_python_xyz")
	r := newCrawl4AIRunner()
	res, err := r.run(context.Background(), "https://example.com", 30*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if res.Status != "error" {
		t.Errorf("status = %q, want error", res.Status)
	}
	if res.Error == "" {
		t.Error("expected error message for missing python")
	}
}

func TestCrawl4AIRunner_Timeout(t *testing.T) {
	script := `import sys, json, time
time.sleep(10)
`
	wrapper := fakeCrawl4AIWrapper(t, script)
	t.Setenv("EXTRACTION_CRAWL4AI_WRAPPER", wrapper)
	r := newCrawl4AIRunner()
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	res, err := r.run(ctx, "https://example.com", 30*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if res.Status != "error" {
		t.Errorf("status = %q, want error", res.Status)
	}
	if res.Error == "" || !strings.Contains(res.Error, "timed out") {
		t.Errorf("expected timeout error, got %q", res.Error)
	}
}
