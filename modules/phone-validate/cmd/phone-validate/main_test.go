package main

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

// TestRun_FullContract exercises the full stdin→stdout contract: a lead record
// with a valid phone and unrelated fields goes in, the same record plus a
// "phone_validate" key comes out, raw fields are preserved untouched, and two
// audit lines (one per source) are written to stderr. No API key is set, so
// numverify is skipped. Fully offline.
func TestRun_FullContract(t *testing.T) {
	t.Setenv("NUMVERIFY_API_KEY", "")
	in := strings.NewReader(`{"name":"Jane","phone":"+14152007986","company":"Acme"}`)
	var out, errb bytes.Buffer

	if err := run(in, &out, &errb); err != nil {
		t.Fatalf("run returned error: %v", err)
	}

	var lead map[string]interface{}
	if err := json.Unmarshal(out.Bytes(), &lead); err != nil {
		t.Fatalf("stdout is not valid JSON: %v\n%s", err, out.String())
	}

	// Raw fields preserved.
	if lead["name"] != "Jane" || lead["company"] != "Acme" || lead["phone"] != "+14152007986" {
		t.Errorf("raw fields not preserved: %+v", lead)
	}

	pv, ok := lead[resultKey].(map[string]interface{})
	if !ok {
		t.Fatalf("%q key missing or wrong type: %+v", resultKey, lead[resultKey])
	}
	if pv["status"] != "ok" {
		t.Errorf("phone_validate.status = %v, want ok", pv["status"])
	}
	if pv["country"] != "US" {
		t.Errorf("phone_validate.country = %v, want US", pv["country"])
	}
	if pv["is_valid_number"] != true {
		t.Errorf("phone_validate.is_valid_number = %v, want true", pv["is_valid_number"])
	}
	nv, ok := pv["numverify"].(map[string]interface{})
	if !ok || nv["status"] != "skipped" {
		t.Errorf("numverify.status = %v, want skipped", pv["numverify"])
	}

	// Exactly two audit lines on stderr, both JSON, redacted phone (no raw PII).
	lines := nonEmptyLines(errb.String())
	if len(lines) != 2 {
		t.Fatalf("expected 2 audit lines on stderr, got %d:\n%s", len(lines), errb.String())
	}
	for _, ln := range lines {
		var audit map[string]interface{}
		if err := json.Unmarshal([]byte(ln), &audit); err != nil {
			t.Errorf("audit line is not JSON: %v (%q)", err, ln)
		}
		if audit["phone"] == "+14152007986" {
			t.Errorf("audit leaked raw phone number; want redacted, got %v", audit["phone"])
		}
		if audit["legal_basis"] == "" || audit["legal_basis"] == nil {
			t.Errorf("audit missing legal_basis: %v", audit)
		}
	}
}

// TestRun_MissingPhone confirms the CLI still emits a well-formed record (exit
// 0 behavior) when the lead has no phone field — status "unknown", audit lines
// present.
func TestRun_MissingPhone(t *testing.T) {
	t.Setenv("NUMVERIFY_API_KEY", "")
	in := strings.NewReader(`{"name":"NoPhone"}`)
	var out, errb bytes.Buffer

	if err := run(in, &out, &errb); err != nil {
		t.Fatalf("run returned error: %v", err)
	}
	var lead map[string]interface{}
	if err := json.Unmarshal(out.Bytes(), &lead); err != nil {
		t.Fatalf("stdout not JSON: %v", err)
	}
	pv := lead[resultKey].(map[string]interface{})
	if pv["status"] != "unknown" {
		t.Errorf("status = %v, want unknown", pv["status"])
	}
	if len(nonEmptyLines(errb.String())) != 2 {
		t.Errorf("expected 2 audit lines, got:\n%s", errb.String())
	}
}

// TestRun_BadJSON confirms the only non-zero-exit condition: unreadable input.
func TestRun_BadJSON(t *testing.T) {
	in := strings.NewReader(`{not json`)
	var out, errb bytes.Buffer
	if err := run(in, &out, &errb); err == nil {
		t.Fatalf("expected an error for malformed stdin JSON, got nil")
	}
}

func nonEmptyLines(s string) []string {
	var out []string
	for _, ln := range strings.Split(s, "\n") {
		if strings.TrimSpace(ln) != "" {
			out = append(out, ln)
		}
	}
	return out
}
