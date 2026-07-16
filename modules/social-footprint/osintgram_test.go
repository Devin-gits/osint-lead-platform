package socialfootprint

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// fakeOsintgramHome writes a fake main.py into a temp dir and returns the path
// that should be assigned to SOCIAL_FOOTPRINT_OSINTGRAM_HOME.
func fakeOsintgramHome(t *testing.T, mainScript string) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "main.py"), []byte(mainScript), 0o755); err != nil {
		t.Fatal(err)
	}
	return dir
}

// runWrapper invokes the real osintgram wrapper with the supplied env and args and
// returns its stdout. It skips if python3 is not available.
func runWrapper(t *testing.T, env map[string]string, args ...string) (string, int) {
	t.Helper()
	python, err := exec.LookPath("python3")
	if err != nil {
		t.Skip("python3 not available")
	}

	wrapper, err := locateOsintgramWrapper()
	if err != nil {
		t.Fatalf("could not locate wrapper: %v", err)
	}

	cmd := exec.Command(python, append([]string{wrapper}, args...)...)
	for k, v := range env {
		cmd.Env = append(os.Environ(), k+"="+v)
	}
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	err = cmd.Run()
	exitCode := 0
	if exitErr, ok := err.(*exec.ExitError); ok {
		exitCode = exitErr.ExitCode()
	}
	return out.String(), exitCode
}

func TestOsintgramWrapper_CommandAllowlist(t *testing.T) {
	// If the allowlist failed, this main.py would print "EXECUTED" and exit 1.
	home := fakeOsintgramHome(t, `import sys
print("EXECUTED")
sys.exit(1)
`)
	env := map[string]string{"SOCIAL_FOOTPRINT_OSINTGRAM_HOME": home}
	stdout, _ := runWrapper(t, env, "--handle", "x", "--command", "photos")
	if strings.Contains(stdout, "EXECUTED") {
		t.Fatal("banned command still executed Osintgram main.py")
	}
	var out wrapperOutput
	if err := json.Unmarshal([]byte(stdout), &out); err != nil {
		t.Fatalf("wrapper output is not JSON: %v\n%s", err, stdout)
	}
	if !strings.Contains(out.Error, "not allowed") {
		t.Errorf("expected allowlist rejection, got error=%q", out.Error)
	}
	if len(out.Results) != 0 {
		t.Errorf("expected no results for rejected command, got %d", len(out.Results))
	}
}

func TestOsintgramWrapper_MissingHome(t *testing.T) {
	env := map[string]string{"SOCIAL_FOOTPRINT_OSINTGRAM_HOME": ""}
	stdout, _ := runWrapper(t, env, "--handle", "x", "--command", "info")
	var out wrapperOutput
	if err := json.Unmarshal([]byte(stdout), &out); err != nil {
		t.Fatalf("wrapper output is not JSON: %v\n%s", err, stdout)
	}
	if !strings.Contains(out.Error, "SOCIAL_FOOTPRINT_OSINTGRAM_HOME") {
		t.Errorf("expected missing-home error, got %q", out.Error)
	}
}

func TestOsintgramWrapper_NotFound(t *testing.T) {
	home := fakeOsintgramHome(t, `import sys
print("Oops... x non exist, please enter a valid username.")
sys.exit(2)
`)
	env := map[string]string{"SOCIAL_FOOTPRINT_OSINTGRAM_HOME": home}
	stdout, _ := runWrapper(t, env, "--handle", "x", "--command", "info")
	var out wrapperOutput
	if err := json.Unmarshal([]byte(stdout), &out); err != nil {
		t.Fatalf("wrapper output is not JSON: %v\n%s", err, stdout)
	}
	if len(out.Results) != 1 || out.Results[0].Status != "available" {
		t.Errorf("expected available result for not-found, got %+v", out.Results)
	}
	if out.Results[0].HTTPStatus != 404 {
		t.Errorf("expected http_status 404 for not-found, got %d", out.Results[0].HTTPStatus)
	}
}

func TestOsintgramWrapper_ClaimedJSON(t *testing.T) {
	home := fakeOsintgramHome(t, `import sys, json, os, argparse
p = argparse.ArgumentParser()
p.add_argument("id")
p.add_argument("--command")
p.add_argument("--json", action="store_true")
p.add_argument("--output")
args = p.parse_args()
# Simulate Osintgram writing the info JSON under <output>/<id>/<id>_info.json.
out_dir = os.path.join(args.output, args.id)
os.makedirs(out_dir, exist_ok=True)
info = {
    "pk": 123456789,
    "full_name": "Nat Geo",
    "biography": "This text must NOT be surfaced in the platform signal.",
    "follower_count": 150000,
    "following_count": 45,
    "media_count": 1200,
    "is_private": False,
    "is_verified": True,
    "is_business": True,
    "public_email": "press@natgeo.com",
}
with open(os.path.join(out_dir, args.id + "_info.json"), "w") as f:
    json.dump(info, f)
`)
	env := map[string]string{"SOCIAL_FOOTPRINT_OSINTGRAM_HOME": home}
	stdout, _ := runWrapper(t, env, "--handle", "natgeo", "--command", "info")
	var out wrapperOutput
	if err := json.Unmarshal([]byte(stdout), &out); err != nil {
		t.Fatalf("wrapper output is not JSON: %v\n%s", err, stdout)
	}
	if out.Error != "" {
		t.Errorf("unexpected wrapper error: %s", out.Error)
	}
	if len(out.Results) != 1 || out.Results[0].Status != "claimed" {
		t.Fatalf("expected claimed result, got %+v", out.Results)
	}
	ig := out.Results[0].Instagram
	if ig == nil {
		t.Fatal("expected InstagramDetails, got nil")
	}
	if ig.UserID != "123456789" {
		t.Errorf("user_id = %q, want 123456789", ig.UserID)
	}
	if ig.IsPrivate || !ig.IsVerified || !ig.IsBusiness {
		t.Errorf("flags mismatch: private=%v verified=%v business=%v", ig.IsPrivate, ig.IsVerified, ig.IsBusiness)
	}
	if ig.FollowerCount != 150000 || ig.FollowingCount != 45 || ig.MediaCount != 1200 {
		t.Errorf("counts mismatch: %+v", ig)
	}
	if !ig.HasPublicEmail {
		t.Error("expected HasPublicEmail true")
	}
	// Public email string itself must not be present in the normalized output.
	if strings.Contains(stdout, "press@natgeo.com") {
		t.Error("wrapper output leaked the raw public email string")
	}
}

func TestOsintgramWrapper_LoginChallengeIsUnknown(t *testing.T) {
	// A login/challenge failure must degrade to unknown, never to available.
	home := fakeOsintgramHome(t, `import sys
print("challenge_required: please check your account")
sys.exit(2)
`)
	env := map[string]string{"SOCIAL_FOOTPRINT_OSINTGRAM_HOME": home}
	stdout, _ := runWrapper(t, env, "--handle", "x", "--command", "info")
	var out wrapperOutput
	if err := json.Unmarshal([]byte(stdout), &out); err != nil {
		t.Fatalf("wrapper output is not JSON: %v\n%s", err, stdout)
	}
	if len(out.Results) != 0 {
		t.Fatalf("expected no results for login/challenge failure, got %+v", out.Results)
	}
	if out.Error == "" {
		t.Error("expected error note for login/challenge failure")
	}
}

func TestOsintgramWrapper_AmbiguousNotFoundPhraseIsUnknown(t *testing.T) {
	// Generic "not found" phrases without the explicit user-not-found exit code
	// and marker must be treated as unknown.
	home := fakeOsintgramHome(t, `import sys
print("page not found: internal error")
sys.exit(1)
`)
	env := map[string]string{"SOCIAL_FOOTPRINT_OSINTGRAM_HOME": home}
	stdout, _ := runWrapper(t, env, "--handle", "x", "--command", "info")
	var out wrapperOutput
	if err := json.Unmarshal([]byte(stdout), &out); err != nil {
		t.Fatalf("wrapper output is not JSON: %v\n%s", err, stdout)
	}
	if len(out.Results) != 0 {
		t.Fatalf("expected unknown (no results) for ambiguous failure, got %+v", out.Results)
	}
	if out.Error == "" {
		t.Error("expected error note for ambiguous failure")
	}
}

func TestOsintgramWrapper_CredentialsAndTokenRedacted(t *testing.T) {
	home := fakeOsintgramHome(t, `import sys
print("some error")
sys.exit(1)
`)
	dir := t.TempDir()
	creds := filepath.Join(dir, "credentials.ini")
	secret := "ig_password_12345_super_secret"
	if err := os.WriteFile(creds, []byte("[Credentials]\nusername=test\npassword="+secret+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	env := map[string]string{
		"SOCIAL_FOOTPRINT_OSINTGRAM_HOME":       home,
		"SOCIAL_FOOTPRINT_OSINTGRAM_CREDENTIALS": creds,
		"HIKERAPI_TOKEN":                        "hiker_secret_token_xyz",
	}
	stdout, _ := runWrapper(t, env, "--handle", "x", "--command", "info")
	combined := stdout // wrapper captures its own stderr into stdout for this helper
	for _, s := range []string{secret, "hiker_secret_token_xyz"} {
		if strings.Contains(combined, s) {
			t.Errorf("wrapper output leaked secret %q", s)
		}
	}
	var out wrapperOutput
	if err := json.Unmarshal([]byte(stdout), &out); err != nil {
		t.Fatalf("wrapper output is not JSON: %v\n%s", err, stdout)
	}
	if out.Error == "" {
		t.Error("expected error note for subprocess failure")
	}
}

func TestOsintgramRunner_ClaimedFromFakeWrapper(t *testing.T) {
	home := fakeOsintgramHome(t, "# placeholder main.py")
	t.Setenv("SOCIAL_FOOTPRINT_OSINTGRAM_HOME", home)

	dir := t.TempDir()
	fake := filepath.Join(dir, "fake_osintgram_wrapper.py")
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
	if err := os.WriteFile(fake, []byte(script), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("SOCIAL_FOOTPRINT_OSINTGRAM_WRAPPER", fake)
	t.Setenv("SOCIAL_FOOTPRINT_OSINTGRAM_PYTHON", "python3")

	r := &osintgramRunner{}
	out, err := r.run(context.Background(), "natgeo", []string{"Instagram"}, 30*time.Second)
	if err != nil {
		t.Fatalf("runner returned error: %v", err)
	}
	if out.Results[0].Status != "claimed" {
		t.Errorf("status = %q, want claimed", out.Results[0].Status)
	}
	if out.Results[0].Instagram == nil || !out.Results[0].Instagram.IsVerified {
		t.Errorf("Instagram details missing or incorrect: %+v", out.Results[0].Instagram)
	}
}

func TestOsintgramRunner_MissingHome(t *testing.T) {
	t.Setenv("SOCIAL_FOOTPRINT_OSINTGRAM_HOME", "")
	// Make sure wrapper can still be located so we exercise the home check.
	if _, err := locateOsintgramWrapper(); err != nil {
		t.Skip("wrapper not locatable: " + err.Error())
	}
	r := &osintgramRunner{}
	_, err := r.run(context.Background(), "natgeo", []string{"Instagram"}, 30*time.Second)
	if err == nil {
		t.Fatal("expected error for missing OSINTGRAM_HOME")
	}
	if !strings.Contains(err.Error(), "SOCIAL_FOOTPRINT_OSINTGRAM_HOME") {
		t.Errorf("expected error to mention OSINTGRAM_HOME, got %v", err)
	}
}

func TestCheck_OsintgramBackendMetadata(t *testing.T) {
	runner := &fakeRunner{byHandle: map[string]wrapperOutput{
		"natgeo": {Results: []platformResult{
			{
				Platform:   "Instagram",
				Status:     "claimed",
				URL:        "https://www.instagram.com/natgeo/",
				HTTPStatus: 200,
				Instagram: &InstagramDetails{
					UserID:     "123",
					IsPrivate:  false,
					IsVerified: true,
					CheckedVia: "osintgram-cli",
				},
			},
		}},
	}}
	v := newTestValidatorWithBackend(runner, BackendOsintgram, []string{"Instagram"})

	res, audits := v.Check(map[string]interface{}{"email": "natgeo@example.com"})

	if res.SourceTool != SourceToolOsintgram {
		t.Errorf("SourceTool = %q, want %q", res.SourceTool, SourceToolOsintgram)
	}
	if got := res.Metadata["platform_count"]; got != 1 {
		t.Errorf("platform_count = %v, want 1", got)
	}
	if res.Confidence != 1.0 {
		t.Errorf("confidence = %f, want 1.0", res.Confidence)
	}
	if res.ActiveSignals != 1 {
		t.Errorf("active_signals = %d, want 1", res.ActiveSignals)
	}
	if len(audits) != 1 || audits[0].LegalBasis != LegalBasis {
		t.Errorf("expected one audit with legal basis, got %+v", audits)
	}
}

func TestNewValidator_OsintgramDefaultMinInterval(t *testing.T) {
	// Ensure no env override leaks from a previous test.
	t.Setenv("SOCIAL_FOOTPRINT_MIN_INTERVAL", "")
	v := NewValidatorWithBackend(0, 0, BackendOsintgram)
	if v.backend != BackendOsintgram {
		t.Fatalf("backend = %q, want osintgram", v.backend)
	}
	if v.limiter.minInterval != DefaultOsintgramMinInterval {
		t.Errorf("minInterval = %v, want %v", v.limiter.minInterval, DefaultOsintgramMinInterval)
	}
}

func TestNewValidator_OsintgramHonorsExplicitMinInterval(t *testing.T) {
	v := NewValidatorWithBackend(0, 3*time.Second, BackendOsintgram)
	if v.limiter.minInterval != 3*time.Second {
		t.Errorf("minInterval = %v, want 3s", v.limiter.minInterval)
	}
}

// TestOsintgramRunner_Live runs against a real Osintgram install if the operator
// opts in with SOCIAL_FOOTPRINT_LIVE_OSINTGRAM=1 and points to a configured
// SOCIAL_FOOTPRINT_OSINTGRAM_HOME. It is skipped in short mode and by default.
func TestOsintgramRunner_Live(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping live Osintgram test in short mode")
	}
	if os.Getenv("SOCIAL_FOOTPRINT_LIVE_OSINTGRAM") != "1" {
		t.Skip("SOCIAL_FOOTPRINT_LIVE_OSINTGRAM=1 not set")
	}
	home := os.Getenv("SOCIAL_FOOTPRINT_OSINTGRAM_HOME")
	if home == "" {
		t.Skip("SOCIAL_FOOTPRINT_OSINTGRAM_HOME not set")
	}
	r := &osintgramRunner{}
	out, err := r.run(context.Background(), "natgeo", []string{"Instagram"}, 120*time.Second)
	if err != nil {
		t.Fatalf("live Osintgram run errored: %v", err)
	}
	if len(out.Results) == 0 {
		t.Fatal("expected at least one result from live Osintgram run")
	}
}
