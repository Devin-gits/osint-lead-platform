package main

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

// TestRun_EndToEnd drives the CLI exactly as the pipeline does: a lead record
// in on stdin, an augmented record out on stdout, one audit line per tool on
// stderr. It uses a real domain so both sub-tools actually run (requires
// network; theHarvester degrades to "unknown" if not installed).
func TestRun_EndToEnd(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping network + subprocess test in -short mode")
	}
	in := strings.NewReader(`{"company":"OWASP","domain":"owasp.org","email":"info@owasp.org"}`)
	var out, errOut bytes.Buffer

	if err := run(in, &out, &errOut); err != nil {
		t.Fatalf("run returned error: %v", err)
	}

	var lead map[string]interface{}
	if err := json.Unmarshal(out.Bytes(), &lead); err != nil {
		t.Fatalf("stdout is not valid JSON: %v\n%s", err, out.String())
	}

	// Raw ingested fields must be preserved untouched.
	if lead["company"] != "OWASP" || lead["domain"] != "owasp.org" || lead["email"] != "info@owasp.org" {
		t.Errorf("raw fields altered or dropped: %+v", lead)
	}

	di, ok := lead[resultKey].(map[string]interface{})
	if !ok {
		t.Fatalf("missing %q key in output: %+v", resultKey, lead)
	}
	wc, ok := di["web_check"].(map[string]interface{})
	if !ok {
		t.Fatalf("missing web_check sub-result: %+v", di)
	}
	if wc["status"] != "ok" {
		t.Errorf("web_check.status = %v, want ok", wc["status"])
	}
	if _, ok := di["harvester"].(map[string]interface{}); !ok {
		t.Fatalf("missing harvester sub-result: %+v", di)
	}
	if tools, ok := di["source_tools"].([]interface{}); !ok || len(tools) != 2 {
		t.Errorf("source_tools = %v, want 2 entries", di["source_tools"])
	}

	// One audit line per tool on stderr (two lines), each valid JSON.
	lines := strings.Split(strings.TrimSpace(errOut.String()), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 audit lines on stderr, got %d:\n%s", len(lines), errOut.String())
	}
	for _, ln := range lines {
		var audit map[string]interface{}
		if err := json.Unmarshal([]byte(ln), &audit); err != nil {
			t.Fatalf("stderr audit line is not valid JSON: %v\n%s", err, ln)
		}
		if audit["tool"] == "" || audit["legal_basis"] == "" {
			t.Errorf("audit line missing tool/legal_basis: %+v", audit)
		}
	}
}

// TestRun_BadInput confirms malformed stdin is a hard error (non-nil), distinct
// from a sub-tool failure (which stays in-band).
func TestRun_BadInput(t *testing.T) {
	var out, errOut bytes.Buffer
	if err := run(strings.NewReader("{not json"), &out, &errOut); err == nil {
		t.Fatal("run accepted malformed JSON, want error")
	}
}
