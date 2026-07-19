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

	in := strings.NewReader(`{"url":"https://example.com","permission_ref":"CAMP-2026-Q3-001","company":"Acme"}`)
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
	if audit["request_url"] != "https://example.com" {
		t.Errorf("audit request_url = %v", audit["request_url"])
	}
	if audit["permission_ref"] != "CAMP-2026-Q3-001" {
		t.Errorf("audit permission_ref = %v", audit["permission_ref"])
	}
	if audit["legal_basis"] == "" {
		t.Error("audit missing legal_basis")
	}
	if audit["module"] != "extraction" {
		t.Errorf("audit module = %v", audit["module"])
	}
	if audit["limits"] == "" {
		t.Error("audit missing limits")
	}
	if audit["tool_version"] == "" {
		t.Error("audit missing tool_version")
	}
}

func TestRun_MissingURL(t *testing.T) {
	in := strings.NewReader(`{"company":"Acme","permission_ref":"T-1"}`)
	var out, errOut bytes.Buffer
	if err := run(in, &out, &errOut); err == nil {
		t.Fatal("expected error for missing URL")
	}
}

func TestRun_MissingPermissionRef(t *testing.T) {
	in := strings.NewReader(`{"url":"https://example.com","company":"Acme"}`)
	var out, errOut bytes.Buffer
	if err := run(in, &out, &errOut); err != nil {
		t.Fatalf("run returned error: %v\nstderr: %s", err, errOut.String())
	}
	var lead map[string]interface{}
	if err := json.Unmarshal(out.Bytes(), &lead); err != nil {
		t.Fatalf("stdout not JSON: %v\n%s", err, out.String())
	}
	ex, ok := lead[resultKey].(map[string]interface{})
	if !ok {
		t.Fatalf("missing %q key", resultKey)
	}
	if ex["status"] != "skipped" {
		t.Errorf("status = %v, want skipped", ex["status"])
	}
	var audit map[string]interface{}
	if err := json.Unmarshal(errOut.Bytes(), &audit); err != nil {
		t.Fatalf("stderr audit not JSON: %v\n%s", err, errOut.String())
	}
	if audit["status"] != "skipped" {
		t.Errorf("audit status = %v", audit["status"])
	}
}

func TestRun_BadInput(t *testing.T) {
	var out, errOut bytes.Buffer
	if err := run(strings.NewReader("{not json"), &out, &errOut); err == nil {
		t.Fatal("expected error for malformed JSON")
	}
}

func TestRun_RejectedPrivateURL(t *testing.T) {
	in := strings.NewReader(`{"url":"http://127.0.0.1/","permission_ref":"T-1"}`)
	var out, errOut bytes.Buffer
	if err := run(in, &out, &errOut); err != nil {
		t.Fatalf("run returned error: %v\nstderr: %s", err, errOut.String())
	}
	var lead map[string]interface{}
	if err := json.Unmarshal(out.Bytes(), &lead); err != nil {
		t.Fatalf("stdout not JSON: %v\n%s", err, out.String())
	}
	ex := lead[resultKey].(map[string]interface{})
	if ex["status"] != "skipped" {
		t.Errorf("status = %v, want skipped", ex["status"])
	}
}
