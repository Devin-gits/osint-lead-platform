package main

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestRun_SkippedNoHandle exercises the full stdin->stdout contract for a lead
// with no derivable handle: the same record comes back with a "social_footprint"
// key whose status is "skipped", raw fields are preserved, and exactly one audit
// line is written to stderr. Fully offline — the Maigret subprocess is never
// invoked on the skip path.
func TestRun_SkippedNoHandle(t *testing.T) {
	in := strings.NewReader(`{"name":"Jane","phone":"+15551234567","company":"Acme"}`)
	var out, errb bytes.Buffer

	if err := run(in, &out, &errb); err != nil {
		t.Fatalf("run returned error: %v", err)
	}

	var lead map[string]interface{}
	if err := json.Unmarshal(out.Bytes(), &lead); err != nil {
		t.Fatalf("stdout is not valid JSON: %v\n%s", err, out.String())
	}
	if lead["name"] != "Jane" || lead["company"] != "Acme" {
		t.Errorf("raw fields not preserved: %+v", lead)
	}
	sf, ok := lead[resultKey].(map[string]interface{})
	if !ok {
		t.Fatalf("%q key missing: %+v", resultKey, lead[resultKey])
	}
	if sf["status"] != "skipped" {
		t.Errorf("status = %v, want skipped", sf["status"])
	}
	if len(nonEmptyLines(errb.String())) != 1 {
		t.Errorf("expected exactly one audit line, got:\n%s", errb.String())
	}
}

// TestRun_FullContractWithFakeWrapper exercises the real subprocess path against
// a fake wrapper script (a tiny Python program that prints the module's JSON
// contract), so the stdin->stdout->audit flow and JSON parsing are covered
// without network or a real Maigret install. It verifies the raw email is never
// leaked into the audit (only the derived handle) and that a claimed signal is
// surfaced.
func TestRun_FullContractWithFakeWrapper(t *testing.T) {
	if _, err := exec.LookPath("python3"); err != nil {
		t.Skip("python3 not available")
	}
	dir := t.TempDir()
	fake := filepath.Join(dir, "fake_wrapper.py")
	script := `import sys, json
args = sys.argv[1:]
u = args[args.index("--username") + 1]
print(json.dumps({
    "tool": "maigret", "version": "fake", "username": u,
    "sites_requested": ["GitHub"],
    "results": [{"platform": "GitHub", "status": "claimed",
                 "url": "https://github.com/" + u, "http_status": 200}],
    "checked_at": "2026-07-13T00:00:00Z", "error": "",
}))
`
	if err := os.WriteFile(fake, []byte(script), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("SOCIAL_FOOTPRINT_WRAPPER", fake)
	t.Setenv("SOCIAL_FOOTPRINT_MIN_INTERVAL", "0")
	t.Setenv("SOCIAL_FOOTPRINT_TIMEOUT", "20s")

	in := strings.NewReader(`{"email":"jane.smith@acme.com","company":"Acme"}`)
	var out, errb bytes.Buffer
	if err := run(in, &out, &errb); err != nil {
		t.Fatalf("run returned error: %v", err)
	}

	var lead map[string]interface{}
	if err := json.Unmarshal(out.Bytes(), &lead); err != nil {
		t.Fatalf("stdout not JSON: %v\n%s", err, out.String())
	}
	sf := lead[resultKey].(map[string]interface{})
	if sf["status"] != "ok" {
		t.Errorf("status = %v, want ok\n%s", sf["status"], out.String())
	}
	if sf["active_signals"].(float64) < 1 {
		t.Errorf("active_signals = %v, want >= 1", sf["active_signals"])
	}

	for _, ln := range nonEmptyLines(errb.String()) {
		var a map[string]interface{}
		if err := json.Unmarshal([]byte(ln), &a); err != nil {
			t.Errorf("audit line not JSON: %v (%q)", err, ln)
			continue
		}
		if a["handle"] == "jane.smith@acme.com" {
			t.Errorf("audit leaked raw email; want handle only, got %v", a["handle"])
		}
		if a["legal_basis"] == "" || a["legal_basis"] == nil {
			t.Errorf("audit missing legal_basis: %v", a)
		}
	}
}

// TestRun_SpiderFootEnrichment exercises the optional SpiderFoot source using
// the same fake-wrapper technique. It enables SpiderFoot, provides fake wrappers
// for both backends, and verifies the merged output and metadata.
func TestRun_SpiderFootEnrichment(t *testing.T) {
	if _, err := exec.LookPath("python3"); err != nil {
		t.Skip("python3 not available")
	}
	dir := t.TempDir()

	fakeMaigret := filepath.Join(dir, "fake_maigret.py")
	maigretScript := `import sys, json
args = sys.argv[1:]
u = args[args.index("--username") + 1]
print(json.dumps({
    "tool": "maigret", "version": "fake", "username": u,
    "sites_requested": ["GitHub"],
    "results": [{"platform": "GitHub", "status": "claimed",
                 "url": "https://github.com/" + u, "http_status": 200}],
    "checked_at": "2026-07-13T00:00:00Z", "error": "",
}))
`
	if err := os.WriteFile(fakeMaigret, []byte(maigretScript), 0o644); err != nil {
		t.Fatal(err)
	}

	fakeSpider := filepath.Join(dir, "fake_spiderfoot.py")
	spiderScript := `import sys, json
args = sys.argv[1:]
u = args[args.index("--username") + 1]
print(json.dumps({
    "tool": "spiderfoot", "version": "4.0", "username": u,
    "sites_requested": ["Keybase"],
    "results": [{"platform": "Keybase", "status": "claimed",
                 "url": "https://keybase.io/" + u, "http_status": 200}],
    "checked_at": "2026-07-13T00:00:00Z", "error": "",
}))
`
	if err := os.WriteFile(fakeSpider, []byte(spiderScript), 0o644); err != nil {
		t.Fatal(err)
	}

	t.Setenv("SOCIAL_FOOTPRINT_WRAPPER", fakeMaigret)
	t.Setenv("SOCIAL_FOOTPRINT_SPIDERFOOT_ENABLED", "true")
	t.Setenv("SOCIAL_FOOTPRINT_SPIDERFOOT_WRAPPER", fakeSpider)
	t.Setenv("SOCIAL_FOOTPRINT_MIN_INTERVAL", "0")
	t.Setenv("SOCIAL_FOOTPRINT_TIMEOUT", "20s")

	// Use an undotted email so only one handle is derived, making the fake-wrapper
	// math deterministic: Maigret claims GitHub + SpiderFoot claims Keybase = 2.
	in := strings.NewReader(`{"email":"janesmith@acme.com","company":"Acme"}`)
	var out, errb bytes.Buffer
	if err := run(in, &out, &errb); err != nil {
		t.Fatalf("run returned error: %v", err)
	}

	var lead map[string]interface{}
	if err := json.Unmarshal(out.Bytes(), &lead); err != nil {
		t.Fatalf("stdout not JSON: %v\n%s", err, out.String())
	}
	sf := lead[resultKey].(map[string]interface{})
	if sf["status"] != "ok" {
		t.Fatalf("status = %v, want ok\n%s", sf["status"], out.String())
	}
	if sf["active_signals"].(float64) != 2 {
		t.Errorf("active_signals = %v, want 2", sf["active_signals"])
	}

	meta := sf["metadata"].(map[string]interface{})
	if _, ok := meta["spiderfoot_platform_count"]; !ok {
		t.Errorf("metadata missing spiderfoot_platform_count: %+v", meta)
	}
	src, _ := meta["source_tool_spiderfoot"].(string)
	if src == "" {
		t.Errorf("metadata missing source_tool_spiderfoot: %+v", meta)
	}
}

func TestRun_BadJSON(t *testing.T) {
	in := strings.NewReader(`{not json`)
	var out, errb bytes.Buffer
	if err := run(in, &out, &errb); err == nil {
		t.Fatal("expected error for malformed stdin JSON, got nil")
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
