package main

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

// TestRun_EndToEnd drives the CLI exactly as the pipeline does: a lead record
// in on stdin, an augmented record out on stdout, an audit line on stderr. It
// uses a real email so the underlying verifier actually runs (requires network).
func TestRun_EndToEnd(t *testing.T) {
	in := strings.NewReader(`{"name":"Jane","email":"support@github.com","company":"Acme"}`)
	var out, errOut bytes.Buffer

	if err := run(in, &out, &errOut); err != nil {
		t.Fatalf("run returned error: %v", err)
	}

	var lead map[string]interface{}
	if err := json.Unmarshal(out.Bytes(), &lead); err != nil {
		t.Fatalf("stdout is not valid JSON: %v\n%s", err, out.String())
	}

	// Raw ingested fields must be preserved untouched.
	if lead["name"] != "Jane" || lead["company"] != "Acme" || lead["email"] != "support@github.com" {
		t.Errorf("raw fields altered or dropped: %+v", lead)
	}

	ev, ok := lead[resultKey].(map[string]interface{})
	if !ok {
		t.Fatalf("missing %q key in output: %+v", resultKey, lead)
	}
	if ev["status"] != "ok" {
		t.Errorf("email_validate.status = %v, want ok", ev["status"])
	}
	if ev["syntax_valid"] != true {
		t.Errorf("email_validate.syntax_valid = %v, want true", ev["syntax_valid"])
	}

	// An audit line must be emitted on stderr for every call.
	var audit map[string]interface{}
	if err := json.Unmarshal(bytes.TrimSpace(errOut.Bytes()), &audit); err != nil {
		t.Fatalf("stderr audit line is not valid JSON: %v\n%s", err, errOut.String())
	}
	if audit["tool"] == "" || audit["legal_basis"] == "" {
		t.Errorf("audit line missing tool/legal_basis: %+v", audit)
	}
}

// TestRun_BadInput confirms malformed stdin is a hard error (non-nil), distinct
// from a validation failure (which stays in-band).
func TestRun_BadInput(t *testing.T) {
	var out, errOut bytes.Buffer
	if err := run(strings.NewReader("{not json"), &out, &errOut); err == nil {
		t.Fatal("run accepted malformed JSON, want error")
	}
}
