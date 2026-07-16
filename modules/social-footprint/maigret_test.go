package socialfootprint

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestParseWrapperOutput_CleanJSON(t *testing.T) {
	want := wrapperOutput{
		Tool:           "maigret",
		Version:        "0.6.2",
		Username:       "soxoj",
		SitesRequested: []string{"GitHub"},
		Results: []platformResult{
			{Platform: "GitHub", Status: "claimed", URL: "https://github.com/soxoj", HTTPStatus: 200},
		},
		CheckedAt: "2026-07-13T00:00:00Z",
		Error:     "",
	}
	b, _ := json.Marshal(want)
	got, err := parseWrapperOutput(string(b))
	if err != nil {
		t.Fatalf("parseWrapperOutput error = %v", err)
	}
	if got.Tool != want.Tool || got.Username != want.Username || len(got.Results) != 1 {
		t.Errorf("parsed %+v, want %+v", got, want)
	}
}

func TestParseWrapperOutput_JSONWithNoise(t *testing.T) {
	payload, _ := json.Marshal(wrapperOutput{
		Tool:     "maigret",
		Username: "soxoj",
		Results:  []platformResult{{Platform: "GitHub", Status: "claimed", HTTPStatus: 200}},
	})
	stdout := "some log line\n" + string(payload) + "\nmore trailing noise"
	got, err := parseWrapperOutput(stdout)
	if err != nil {
		t.Fatalf("parseWrapperOutput error = %v", err)
	}
	if got.Username != "soxoj" || len(got.Results) != 1 {
		t.Errorf("parsed %+v, want one GitHub result for soxoj", got)
	}
}

func TestParseWrapperOutput_PrettyPrintedJSON(t *testing.T) {
	pretty := `{
  "tool": "maigret",
  "username": "soxoj",
  "sites_requested": ["GitHub"],
  "results": [
    {"platform": "GitHub", "status": "claimed", "url": "https://github.com/soxoj", "http_status": 200}
  ],
  "checked_at": "2026-07-13T00:00:00Z",
  "error": ""
}`
	got, err := parseWrapperOutput("warning: something\n" + pretty + "\nother stuff")
	if err != nil {
		t.Fatalf("parseWrapperOutput error = %v", err)
	}
	if got.Username != "soxoj" {
		t.Errorf("username = %q, want soxoj", got.Username)
	}
	if len(got.Results) != 1 || got.Results[0].Platform != "GitHub" {
		t.Errorf("results = %+v", got.Results)
	}
}

func TestParseWrapperOutput_Empty(t *testing.T) {
	_, err := parseWrapperOutput("")
	if err == nil {
		t.Fatal("expected error for empty output")
	}
}

func TestParseWrapperOutput_InvalidJSON(t *testing.T) {
	_, err := parseWrapperOutput("not json")
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestLocateWrapper_EnvOverride(t *testing.T) {
	dir := t.TempDir()
	wrapperPath := filepath.Join(dir, "maigret_check.py")
	if err := os.WriteFile(wrapperPath, []byte("#!/usr/bin/env python3\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv(wrapperEnv, wrapperPath)
	got, err := locateWrapper()
	if err != nil {
		t.Fatalf("locateWrapper error = %v", err)
	}
	if got != wrapperPath {
		t.Errorf("locateWrapper = %q, want %q", got, wrapperPath)
	}
}

func TestLocateWrapper_EnvOverrideMissing(t *testing.T) {
	t.Setenv(wrapperEnv, "/nonexistent/wrapper.py")
	_, err := locateWrapper()
	if err == nil {
		t.Fatal("expected error when env override does not exist")
	}
}

func TestLocateWrapper_FromWorkingDir(t *testing.T) {
	dir := t.TempDir()
	wrapperDir := filepath.Join(dir, "wrapper")
	if err := os.MkdirAll(wrapperDir, 0o755); err != nil {
		t.Fatal(err)
	}
	wrapperPath := filepath.Join(wrapperDir, "maigret_check.py")
	if err := os.WriteFile(wrapperPath, []byte("#!/usr/bin/env python3\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	old, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(old)

	got, err := locateWrapper()
	if err != nil {
		t.Fatalf("locateWrapper error = %v", err)
	}
	if got != wrapperPath {
		t.Errorf("locateWrapper = %q, want %q", got, wrapperPath)
	}
}

func TestSubprocessRunner_FakeWrapper(t *testing.T) {
	dir := t.TempDir()
	script := filepath.Join(dir, "fake_maigret.py")
	payload, _ := json.Marshal(wrapperOutput{
		Tool:           "maigret",
		Version:        "fake",
		Username:       "fakeuser",
		SitesRequested: []string{"GitHub"},
		Results: []platformResult{
			{Platform: "GitHub", Status: "available", URL: "https://github.com/fakeuser", HTTPStatus: 404},
		},
		CheckedAt: "2026-07-13T00:00:00Z",
		Error:     "",
	})
	body := "import sys, json\n" +
		"args = sys.argv[1:]\n" +
		"u = args[args.index('--username') + 1]\n" +
		"print(json.dumps(" + string(payload) + "))\n"
	if err := os.WriteFile(script, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}

	t.Setenv("SOCIAL_FOOTPRINT_WRAPPER", script)
	t.Setenv("SOCIAL_FOOTPRINT_PYTHON", "python3")

	r := &subprocessRunner{}
	ctx := context.Background()
	out, err := r.run(ctx, "fakeuser", []string{"GitHub"}, 30*time.Second)
	if err != nil {
		t.Fatalf("subprocessRunner.run error = %v", err)
	}
	if out.Username != "fakeuser" {
		t.Errorf("username = %q, want fakeuser", out.Username)
	}
	if len(out.Results) != 1 || out.Results[0].Platform != "GitHub" {
		t.Errorf("unexpected results: %+v", out.Results)
	}
}

func TestSubprocessRunner_Timeout(t *testing.T) {
	dir := t.TempDir()
	script := filepath.Join(dir, "slow.py")
	body := "import time\ntime.sleep(60)\n"
	if err := os.WriteFile(script, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("SOCIAL_FOOTPRINT_WRAPPER", script)

	r := &subprocessRunner{}
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	_, err := r.run(ctx, "fakeuser", []string{"GitHub"}, 200*time.Millisecond)
	if err == nil {
		t.Fatal("expected timeout error")
	}
}

// TestParseWrapperOutput_NestedObjectAfterArray verifies the fallback JSON scanner
// extracts the full top-level object even when it contains an array followed by
// another nested object. The old brace counter treated '[' and ']' as object
// boundaries, which could either cause an early return at the nested object or
// overshoot the real top-level '}' after depth went negative.
func TestParseWrapperOutput_NestedObjectAfterArray(t *testing.T) {
	// results is an array; metadata is an object after it, then the top-level object
	// closes. A brace counter that mishandles arrays would return the inner object
	// or fail to return at all.
	stdout := `warning: stray log
{
  "tool": "maigret",
  "username": "soxoj",
  "results": [
    {"platform": "GitHub", "status": "claimed", "url": "https://github.com/soxoj", "http_status": 200}
  ],
  "metadata": {"platform_count": 15},
  "checked_at": "2026-07-13T00:00:00Z",
  "error": ""
}
trailing noise`
	got, err := parseWrapperOutput(stdout)
	if err != nil {
		t.Fatalf("parseWrapperOutput error = %v", err)
	}
	if got.Username != "soxoj" {
		t.Errorf("username = %q, want soxoj", got.Username)
	}
	if len(got.Results) != 1 || got.Results[0].Platform != "GitHub" {
		t.Errorf("results = %+v, want one GitHub hit", got.Results)
	}
	// The whole object must be captured, including the trailing metadata field.
	if got.CheckedAt != "2026-07-13T00:00:00Z" {
		t.Errorf("checked_at = %q, want full top-level object to be captured", got.CheckedAt)
	}
}

// TestSubprocessRunner_RealMaigret does one live network check against the real
// embedded Maigret. It is intentionally conservative: it asks for a single
// platform with a username that is highly likely to be "available" on GitHub,
// and skips under -short or when Python/Maigret is unavailable.
func TestSubprocessRunner_RealMaigret(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping live Maigret integration test in short mode")
	}
	if _, err := os.Stat("wrapper/maigret_check.py"); err != nil {
		t.Skip("wrapper/maigret_check.py not present:", err)
	}

	r := &subprocessRunner{}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	out, err := r.run(ctx, "zzzznotrealuser2026abc", []string{"GitHub"}, 60*time.Second)
	if err != nil {
		t.Fatalf("live Maigret run failed: %v", err)
	}
	if out.Error != "" {
		t.Fatalf("wrapper returned error: %s", out.Error)
	}
	if len(out.Results) == 0 {
		t.Fatal("expected at least one platform result from live Maigret run")
	}
	gh := out.Results[0]
	if gh.Platform != "GitHub" {
		t.Errorf("first result platform = %q, want GitHub", gh.Platform)
	}
	if gh.Status != "available" && gh.Status != "claimed" && gh.Status != "unknown" {
		t.Errorf("unexpected status %q for live GitHub result", gh.Status)
	}
}
