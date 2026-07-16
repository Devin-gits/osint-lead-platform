package main

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func TestRun_EndToEnd(t *testing.T) {
	// Use a fake wrapper so the test is network-free.
	dir := t.TempDir()
	script := `import sys, json
print(json.dumps({
    "status": "ok",
    "url": "https://example.com",
    "final_url": "https://example.com/",
    "source_tool": "unclecode/crawl4ai@v0.9.2 (CLI subprocess)",
    "confidence": 0.5,
    "fields": {"company_name":"Acme","emails":["hello@acme.com"],"phones":[],"addresses":[],"social_links":[],"contact_urls":[],"description":"","title":"Home"},
    "raw_markdown": "Acme hello@acme.com",
    "metadata": {"backend":"crawl4ai","http_status":200,"truncated":False,"raw_bytes":19},
    "error": "",
    "checked_at": "2026-07-16T12:00:00Z"
}))
`
	wrapper := dir + "/fake_crawl4ai_extract.py"
	if err := os.WriteFile(wrapper, []byte(script), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("EXTRACTION_CRAWL4AI_WRAPPER", wrapper)
	t.Setenv("EXTRACTION_CRAWL4AI_PYTHON", "python3")

	in := strings.NewReader(`{"url":"https://example.com","company":"Acme"}`)
	var out, errOut bytes.Buffer
	if err := run(in, &out, &errOut); err != nil {
		t.Fatalf("run returned error: %v\nstderr: %s", err, errOut.String())
	}

	var lead map[string]interface{}
	if err := json.Unmarshal(out.Bytes(), &lead); err != nil {
		t.Fatalf("stdout not JSON: %v\n%s", err, out.String())
	}
	if lead["company"] != "Acme" || lead["url"] != "https://example.com" {
		t.Errorf("raw fields not preserved: %+v", lead)
	}
	ex, ok := lead[resultKey].(map[string]interface{})
	if !ok {
		t.Fatalf("missing %q key", resultKey)
	}
	if ex["status"] != "ok" {
		t.Errorf("status = %v, want ok", ex["status"])
	}
	fields, _ := ex["fields"].(map[string]interface{})
	if fields["company_name"] != "Acme" {
		t.Errorf("company_name = %v", fields["company_name"])
	}

	var audit map[string]interface{}
	if err := json.Unmarshal(errOut.Bytes(), &audit); err != nil {
		t.Fatalf("stderr audit not JSON: %v\n%s", err, errOut.String())
	}
	if audit["url"] != "https://example.com" {
		t.Errorf("audit url = %v", audit["url"])
	}
	if audit["legal_basis"] == "" {
		t.Error("audit missing legal_basis")
	}
}

func TestRun_MissingURL(t *testing.T) {
	in := strings.NewReader(`{"company":"Acme"}`)
	var out, errOut bytes.Buffer
	if err := run(in, &out, &errOut); err == nil {
		t.Fatal("expected error for missing URL")
	}
}

func TestRun_BadInput(t *testing.T) {
	var out, errOut bytes.Buffer
	if err := run(strings.NewReader("{not json"), &out, &errOut); err == nil {
		t.Fatal("expected error for malformed JSON")
	}
}
