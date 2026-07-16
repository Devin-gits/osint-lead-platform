package socialfootprint

import (
	"context"
	"fmt"
	"testing"
	"time"
)

// TestBothRunner_UpgradesUnknown verifies the merge rule: Sherlock's "claimed"
// upgrades Maigret's "unknown" for the same platform.
func TestBothRunner_UpgradesUnknown(t *testing.T) {
	primary := &fakeRunner{byHandle: map[string]wrapperOutput{
		"alice": {Results: []platformResult{
			{Platform: "GitHub", Status: "unknown", HTTPStatus: 403},
			{Platform: "Reddit", Status: "claimed", URL: "https://reddit.com/user/alice", HTTPStatus: 200},
		}},
	}}
	secondary := &fakeRunner{byHandle: map[string]wrapperOutput{
		"alice": {Results: []platformResult{
			{Platform: "GitHub", Status: "claimed", URL: "https://github.com/alice", HTTPStatus: 200},
		}},
	}}
	br := &bothRunner{primary: primary, secondary: secondary}

	out, err := br.run(context.Background(), "alice", curatedPlatforms, 5*time.Second)
	if err != nil {
		t.Fatalf("bothRunner.run() error = %v", err)
	}

	byPlatform := map[string]platformResult{}
	for _, r := range out.Results {
		byPlatform[r.Platform] = r
	}

	if byPlatform["GitHub"].Status != "claimed" {
		t.Errorf("GitHub status = %q after merge, want claimed (should have been upgraded from unknown)", byPlatform["GitHub"].Status)
	}
	if byPlatform["Reddit"].Status != "claimed" {
		t.Errorf("Reddit status = %q, want claimed (must not be downgraded)", byPlatform["Reddit"].Status)
	}
}

// TestBothRunner_PrimaryFailsFallsToSherlock verifies that when Maigret errors,
// bothRunner falls back to Sherlock alone.
func TestBothRunner_PrimaryFailsFallsToSherlock(t *testing.T) {
	primary := &fakeRunner{err: fmt.Errorf("maigret wrapper not found")}
	secondary := &fakeRunner{byHandle: map[string]wrapperOutput{
		"alice": {Results: []platformResult{
			{Platform: "GitHub", Status: "claimed", URL: "https://github.com/alice", HTTPStatus: 200},
		}},
	}}
	br := &bothRunner{primary: primary, secondary: secondary}

	out, err := br.run(context.Background(), "alice", curatedPlatforms, 5*time.Second)
	if err != nil {
		t.Fatalf("bothRunner.run() error on fallback = %v", err)
	}
	if len(out.Results) == 0 {
		t.Error("expected results from sherlock fallback")
	}
}

// TestBothRunner_DoesNotDowngradeDefinitive verifies the inverse merge rule:
// Maigret's "claimed" is never overwritten by Sherlock's "available".
func TestBothRunner_DoesNotDowngradeDefinitive(t *testing.T) {
	primary := &fakeRunner{byHandle: map[string]wrapperOutput{
		"alice": {Results: []platformResult{
			{Platform: "GitHub", Status: "claimed", URL: "https://github.com/alice", HTTPStatus: 200},
		}},
	}}
	secondary := &fakeRunner{byHandle: map[string]wrapperOutput{
		"alice": {Results: []platformResult{
			{Platform: "GitHub", Status: "available", HTTPStatus: 404},
		}},
	}}
	br := &bothRunner{primary: primary, secondary: secondary}

	out, err := br.run(context.Background(), "alice", curatedPlatforms, 5*time.Second)
	if err != nil {
		t.Fatalf("bothRunner.run() error = %v", err)
	}
	for _, r := range out.Results {
		if r.Platform == "GitHub" && r.Status != "claimed" {
			t.Errorf("GitHub downgraded from claimed to %q — must never happen", r.Status)
		}
	}
}

// TestNewValidatorWithBackend_SherlockMode verifies sherlockRunner is wired when
// explicitly requested via constructor argument.
func TestNewValidatorWithBackend_SherlockMode(t *testing.T) {
	v := NewValidatorWithBackend(time.Second, 0, BackendSherlock)
	if _, ok := v.runner.(*sherlockRunner); !ok {
		t.Errorf("runner type = %T, want *sherlockRunner", v.runner)
	}
}

// TestNewValidatorWithBackend_BothMode verifies bothRunner is wired for "both".
func TestNewValidatorWithBackend_BothMode(t *testing.T) {
	v := NewValidatorWithBackend(time.Second, 0, BackendBoth)
	if _, ok := v.runner.(*bothRunner); !ok {
		t.Errorf("runner type = %T, want *bothRunner", v.runner)
	}
}

// TestNewValidatorWithBackend_DefaultIsMaigret verifies empty string defaults to Maigret.
func TestNewValidatorWithBackend_DefaultIsMaigret(t *testing.T) {
	v := NewValidatorWithBackend(time.Second, 0, "")
	if _, ok := v.runner.(*subprocessRunner); !ok {
		t.Errorf("runner type = %T, want *subprocessRunner (Maigret default)", v.runner)
	}
}

// TestNewValidatorWithBackend_EnvOverride verifies SOCIAL_FOOTPRINT_BACKEND env
// var overrides the constructor argument.
func TestNewValidatorWithBackend_EnvOverride(t *testing.T) {
	t.Setenv("SOCIAL_FOOTPRINT_BACKEND", "sherlock")
	v := NewValidatorWithBackend(time.Second, 0, BackendMaigret) // arg says maigret
	if _, ok := v.runner.(*sherlockRunner); !ok {
		t.Errorf("env override not respected: runner = %T, want *sherlockRunner", v.runner)
	}
}
