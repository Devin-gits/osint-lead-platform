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

func TestRun_BadJSON(t *testing.T) {
	in := strings.NewReader(`{not json`)
	var out, errb bytes.Buffer
	if err := run(in, &out, &errb); err == nil {
		t.Fatal("expected error for malformed stdin JSON, got nil")
	}
}

// TestRun_FlagUsernameAndEmail verifies that --username and --email are injected
// into the lead map passed to the validator, take precedence over an empty
// stdin, and result in a check against the explicit handle.
func TestRun_FlagUsernameAndEmail(t *testing.T) {
	fake := fakeWrapperPath(t)
	t.Setenv("SOCIAL_FOOTPRINT_WRAPPER", fake)
	t.Setenv("SOCIAL_FOOTPRINT_MIN_INTERVAL", "0")
	t.Setenv("SOCIAL_FOOTPRINT_TIMEOUT", "20s")

	in := strings.NewReader("")
	var out, errb bytes.Buffer
	if err := run(in, &out, &errb, "--username", "direct_user", "--email", "direct@example.com"); err != nil {
		t.Fatalf("run returned error: %v\nstderr: %s", err, errb.String())
	}

	var lead map[string]interface{}
	if err := json.Unmarshal(out.Bytes(), &lead); err != nil {
		t.Fatalf("stdout not JSON: %v\n%s", err, out.String())
	}
	if lead["username"] != "direct_user" {
		t.Errorf("lead[username] = %v, want direct_user", lead["username"])
	}
	if lead["email"] != "direct@example.com" {
		t.Errorf("lead[email] = %v, want direct@example.com", lead["email"])
	}
	sf := lead[resultKey].(map[string]interface{})
	if sf["status"] != "ok" {
		t.Errorf("status = %v, want ok\n%s", sf["status"], out.String())
	}
	if sf["active_signals"].(float64) < 1 {
		t.Errorf("active_signals = %v, want >= 1", sf["active_signals"])
	}
	checked := sf["handles_checked"].([]interface{})
	if len(checked) == 0 || checked[0] != "direct_user" {
		t.Errorf("handles_checked = %v, want direct_user first", checked)
	}
	for _, ln := range nonEmptyLines(errb.String()) {
		var a map[string]interface{}
		if err := json.Unmarshal([]byte(ln), &a); err != nil {
			t.Errorf("audit line not JSON: %v (%q)", err, ln)
			continue
		}
		if a["handle"] == "direct@example.com" {
			t.Errorf("audit leaked raw email; want handle only, got %v", a["handle"])
		}
		if a["legal_basis"] == "" || a["legal_basis"] == nil {
			t.Errorf("audit missing legal_basis: %v", a)
		}
	}
}

// TestRun_FlagsOverrideStdin verifies that CLI flags override the corresponding
// fields in a stdin JSON record while preserving other raw fields.
func TestRun_FlagsOverrideStdin(t *testing.T) {
	fake := fakeWrapperPath(t)
	t.Setenv("SOCIAL_FOOTPRINT_WRAPPER", fake)
	t.Setenv("SOCIAL_FOOTPRINT_MIN_INTERVAL", "0")
	t.Setenv("SOCIAL_FOOTPRINT_TIMEOUT", "20s")

	in := strings.NewReader(`{"username":"stdin_user","email":"stdin@example.com","company":"Acme"}`)
	var out, errb bytes.Buffer
	if err := run(in, &out, &errb, "--username", "flag_user", "--email", "flag@example.com"); err != nil {
		t.Fatalf("run returned error: %v\nstderr: %s", err, errb.String())
	}

	var lead map[string]interface{}
	if err := json.Unmarshal(out.Bytes(), &lead); err != nil {
		t.Fatalf("stdout not JSON: %v\n%s", err, out.String())
	}
	if lead["username"] != "flag_user" {
		t.Errorf("lead[username] = %v, want flag_user", lead["username"])
	}
	if lead["email"] != "flag@example.com" {
		t.Errorf("lead[email] = %v, want flag@example.com", lead["email"])
	}
	if lead["company"] != "Acme" {
		t.Errorf("lead[company] = %v, want Acme (raw field not preserved)", lead["company"])
	}
	sf := lead[resultKey].(map[string]interface{})
	checked := sf["handles_checked"].([]interface{})
	if len(checked) == 0 || checked[0] != "flag_user" {
		t.Errorf("handles_checked = %v, want flag_user first (flags did not take precedence)", checked)
	}
}

// TestRun_OsintgramBackend verifies the CLI path for the optional Osintgram
// backend: the lead record is preserved, the Instagram result is emitted, and the
// audit line carries only the handle (no raw email).
func TestRun_OsintgramBackend(t *testing.T) {
	if _, err := exec.LookPath("python3"); err != nil {
		t.Skip("python3 not available")
	}

	dir := t.TempDir()
	wrapper := filepath.Join(dir, "fake_osintgram_wrapper.py")
	script := `import sys, json
print(json.dumps({
    "tool": "osintgram",
    "version": "fake",
    "username": "natgeo",
    "sites_requested": ["Instagram"],
    "results": [{
        "platform": "Instagram",
        "status": "claimed",
        "url": "https://www.instagram.com/natgeo/",
        "http_status": 200,
        "instagram": {
            "user_id": "123",
            "is_private": False,
            "is_verified": True,
            "is_business": True,
            "follower_count": 100,
            "following_count": 10,
            "media_count": 50,
            "has_public_email": False,
            "checked_via": "osintgram-cli"
        }
    }],
    "checked_at": "2026-07-16T12:00:00Z",
    "error": "",
}))
`
	if err := os.WriteFile(wrapper, []byte(script), 0o644); err != nil {
		t.Fatal(err)
	}

	home := filepath.Join(dir, "osintgram")
	if err := os.MkdirAll(home, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, "main.py"), []byte("# placeholder\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	t.Setenv("SOCIAL_FOOTPRINT_BACKEND", "osintgram")
	t.Setenv("SOCIAL_FOOTPRINT_OSINTGRAM_WRAPPER", wrapper)
	t.Setenv("SOCIAL_FOOTPRINT_OSINTGRAM_HOME", home)
	t.Setenv("SOCIAL_FOOTPRINT_MIN_INTERVAL", "0")

	in := strings.NewReader(`{"email":"natgeo@example.com","company":"Demo"}`)
	var out, errb bytes.Buffer
	if err := run(in, &out, &errb); err != nil {
		t.Fatalf("run returned error: %v\nstderr: %s", err, errb.String())
	}

	var lead map[string]interface{}
	if err := json.Unmarshal(out.Bytes(), &lead); err != nil {
		t.Fatalf("stdout not JSON: %v\n%s", err, out.String())
	}
	if lead["email"] != "natgeo@example.com" || lead["company"] != "Demo" {
		t.Errorf("raw fields not preserved: %+v", lead)
	}
	sf := lead[resultKey].(map[string]interface{})
	if sf["status"] != "ok" {
		t.Errorf("status = %v, want ok\n%s", sf["status"], out.String())
	}
	if sf["active_signals"].(float64) != 1 {
		t.Errorf("active_signals = %v, want 1", sf["active_signals"])
	}
	for _, ln := range nonEmptyLines(errb.String()) {
		var a map[string]interface{}
		if err := json.Unmarshal([]byte(ln), &a); err != nil {
			t.Errorf("audit line not JSON: %v (%q)", err, ln)
			continue
		}
		if a["handle"] == "natgeo@example.com" {
			t.Errorf("audit leaked raw email; want handle only, got %v", a["handle"])
		}
		if a["legal_basis"] == "" || a["legal_basis"] == nil {
			t.Errorf("audit missing legal_basis: %v", a)
		}
	}
}

// fakeWrapperPath writes a tiny Python script that implements the wrapper JSON
// contract and returns its path. Tests can point SOCIAL_FOOTPRINT_WRAPPER at it
// to exercise the subprocess path without a real Maigret install.
func fakeWrapperPath(t *testing.T) string {
	t.Helper()
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
	if err := os.WriteFile(fake, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	return fake
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
