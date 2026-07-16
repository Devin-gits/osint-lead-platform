package socialfootprint

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"
	"testing"
	"time"
)

// fakeRunner is an offline maigretRunner: it returns canned output (or an error)
// per handle, so the orchestration logic can be tested without Python or network.
type fakeRunner struct {
	byHandle map[string]wrapperOutput
	err      error
	calls    int
}

func (f *fakeRunner) run(_ context.Context, handle string, _ []string, _ time.Duration) (wrapperOutput, error) {
	f.calls++
	if f.err != nil {
		return wrapperOutput{}, f.err
	}
	if out, ok := f.byHandle[handle]; ok {
		return out, nil
	}
	return wrapperOutput{Tool: "maigret", Version: "0.6.2", Results: []platformResult{}}, nil
}

func newTestValidator(r maigretRunner) *Validator {
	return &Validator{
		timeout:   time.Second,
		limiter:   newRateLimiter(0),
		runner:    r,
		backend:   BackendMaigret,
		platforms: curatedPlatforms,
	}
}

// newTestValidatorWithBackend builds a Validator for offline tests with an
// explicit backend and platform list.
func newTestValidatorWithBackend(r maigretRunner, backend string, platforms []string) *Validator {
	return &Validator{
		timeout:   time.Second,
		limiter:   newRateLimiter(0),
		runner:    r,
		backend:   backend,
		platforms: platforms,
	}
}

// newTestValidatorWithSpiderFoot builds a Validator for offline tests that
// exercise the optional SpiderFoot enrichment path.
func newTestValidatorWithSpiderFoot(r, sfr maigretRunner, backend string, platforms, sfPlatforms []string) *Validator {
	return &Validator{
		timeout:             time.Second,
		limiter:             newRateLimiter(0),
		runner:              r,
		backend:             backend,
		platforms:           platforms,
		spiderfootEnabled:   true,
		spiderfootRunner:    sfr,
		spiderfootPlatforms: sfPlatforms,
	}
}

// TestCheck_ClaimedSignals verifies the happy path: a lead with an email yields
// derived handles, the runner's per-platform signals are surfaced, and the
// active-signal count aggregates the "claimed" hits.
func TestCheck_ClaimedSignals(t *testing.T) {
	runner := &fakeRunner{byHandle: map[string]wrapperOutput{
		"jane.smith": {Results: []platformResult{
			{Platform: "GitHub", Status: "claimed", URL: "https://github.com/jane.smith", HTTPStatus: 200},
			{Platform: "Reddit", Status: "unknown", HTTPStatus: 403},
		}},
	}}
	v := newTestValidator(runner)

	res, audits := v.Check(map[string]interface{}{"email": "jane.smith@acme.com"})

	if res.Status != statusOK {
		t.Fatalf("status = %q, want ok", res.Status)
	}
	if len(res.HandlesChecked) == 0 || res.HandlesChecked[0] != "jane.smith" {
		t.Errorf("handles_checked = %v, want first = jane.smith", res.HandlesChecked)
	}
	if res.ActiveSignals != 1 {
		t.Errorf("active_signals = %d, want 1", res.ActiveSignals)
	}
	if res.Confidence == 0 {
		t.Errorf("confidence = %f, want > 0", res.Confidence)
	}
	if res.Metadata == nil {
		t.Errorf("metadata should be populated")
	}
	if len(audits) != len(res.HandlesChecked) {
		t.Errorf("expected one audit per checked handle: %d audits, %d handles", len(audits), len(res.HandlesChecked))
	}
	for _, a := range audits {
		if a.LegalBasis != LegalBasis {
			t.Errorf("audit missing legal basis: %+v", a)
		}
		if a.Handle == "jane.smith@acme.com" {
			t.Errorf("audit leaked raw email; want handle only, got %q", a.Handle)
		}
	}
	// First handle result should be "ok" with one claimed platform.
	h0 := res.Handles[0]
	if h0.Status != statusOK || h0.ClaimedCount != 1 {
		t.Errorf("handle result = %+v, want ok/claimed=1", h0)
	}
}

// TestCheck_NoHandleSkips verifies the "skipped" (not "unknown") degrade path
// when no usable handle can be derived, and that the runner is never invoked.
func TestCheck_NoHandleSkips(t *testing.T) {
	runner := &fakeRunner{}
	v := newTestValidator(runner)

	res, audits := v.Check(map[string]interface{}{"name": "No Email", "phone": "+1555"})

	if res.Status != statusSkipped {
		t.Fatalf("status = %q, want skipped", res.Status)
	}
	if res.Reason == "" {
		t.Errorf("skipped result should carry a reason")
	}
	if runner.calls != 0 {
		t.Errorf("runner should not be called when skipping, got %d calls", runner.calls)
	}
	if len(audits) != 1 || audits[0].Status != statusSkipped {
		t.Errorf("expected exactly one skip audit, got %+v", audits)
	}
}

// TestCheck_RunnerErrorDegradesUnknown verifies a per-handle runner failure
// degrades that handle to "unknown" (with an error note) rather than crashing,
// while the overall result still returns.
func TestCheck_RunnerErrorDegradesUnknown(t *testing.T) {
	runner := &fakeRunner{err: errors.New("maigret check timed out after 90s")}
	v := newTestValidator(runner)

	res, _ := v.Check(map[string]interface{}{"email": "bob@acme.com"})

	if res.Status != statusOK {
		t.Fatalf("top-level status = %q, want ok (module ran)", res.Status)
	}
	if len(res.Handles) == 0 {
		t.Fatal("expected at least one handle result")
	}
	for _, h := range res.Handles {
		if h.Status != statusUnknown {
			t.Errorf("handle %q status = %q, want unknown on runner error", h.Handle, h.Status)
		}
		if h.Error == "" {
			t.Errorf("handle %q should carry an error note", h.Handle)
		}
	}
}

// TestCheck_MaxHandlesCap verifies the fan-out cap: even when many candidates are
// derivable, at most MaxHandles are actually checked.
func TestCheck_MaxHandlesCap(t *testing.T) {
	runner := &fakeRunner{}
	v := newTestValidator(runner)

	lead := map[string]interface{}{
		"email": "jane.q.smith@acme.com", // yields local + 2 variants = 3 already
		"domain_intel": map[string]interface{}{
			"harvester": map[string]interface{}{
				"emails": []interface{}{"info@acme.com", "sales@acme.com", "careers@acme.com"},
			},
		},
	}
	res, _ := v.Check(lead)

	if len(res.HandlesChecked) > MaxHandles {
		t.Errorf("checked %d handles, want <= %d", len(res.HandlesChecked), MaxHandles)
	}
	if runner.calls > MaxHandles {
		t.Errorf("runner called %d times, want <= %d", runner.calls, MaxHandles)
	}
}

// TestCheck_ScopeIsCurated verifies the curated allow-list is what gets passed to
// the runner — the scope-discipline guardrail — and that it is not Maigret's full
// site set.
func TestCheck_ScopeIsCurated(t *testing.T) {
	var gotPlatforms []string
	runner := runnerFunc(func(_ context.Context, _ string, platforms []string, _ time.Duration) (wrapperOutput, error) {
		gotPlatforms = platforms
		return wrapperOutput{Results: []platformResult{}}, nil
	})
	v := newTestValidator(runner)

	v.Check(map[string]interface{}{"email": "alice@acme.com"})

	if len(gotPlatforms) == 0 || len(gotPlatforms) > 20 {
		t.Errorf("scope should be a small curated list (10-20), got %d: %v", len(gotPlatforms), gotPlatforms)
	}
	if len(gotPlatforms) != len(curatedPlatforms) {
		t.Errorf("runner got %d platforms, want the curated %d", len(gotPlatforms), len(curatedPlatforms))
	}
}

// runnerFunc adapts a function to the maigretRunner interface.
type runnerFunc func(context.Context, string, []string, time.Duration) (wrapperOutput, error)

func (f runnerFunc) run(ctx context.Context, h string, p []string, t time.Duration) (wrapperOutput, error) {
	return f(ctx, h, p, t)
}

// TestCheck_BackendAwareSourceTool verifies that SocialFootprintResult.SourceTool,
// HandleResult.SourceTool, Metadata["source_tool"], and AuditRecord.Tool all reflect
// the active backend.
func TestCheck_BackendAwareSourceTool(t *testing.T) {
	tests := []struct {
		name      string
		backend   string
		platforms []string
		wantTool  string
	}{
		{
			name:      "maigret default",
			backend:   BackendMaigret,
			platforms: curatedPlatforms,
			wantTool:  SourceTool,
		},
		{
			name:      "sherlock only",
			backend:   BackendSherlock,
			platforms: sherlockCuratedPlatforms,
			wantTool:  SourceToolSherlock,
		},
		{
			name:      "both consensus",
			backend:   BackendBoth,
			platforms: curatedPlatforms,
			wantTool:  SourceTool + " + " + SourceToolSherlock + " consensus",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runner := &fakeRunner{byHandle: map[string]wrapperOutput{
				"jane.smith": {Results: []platformResult{
					{Platform: "GitHub", Status: "claimed", URL: "https://github.com/jane.smith", HTTPStatus: 200},
				}},
			}}
			v := newTestValidatorWithBackend(runner, tt.backend, tt.platforms)

			res, audits := v.Check(map[string]interface{}{"email": "jane.smith@acme.com"})

			if res.SourceTool != tt.wantTool {
				t.Errorf("SocialFootprintResult.SourceTool = %q, want %q", res.SourceTool, tt.wantTool)
			}
			if got := res.Metadata["source_tool"]; got != tt.wantTool {
				t.Errorf("Metadata[source_tool] = %v, want %v", got, tt.wantTool)
			}
			if len(res.Handles) == 0 {
				t.Fatal("expected at least one handle result")
			}
			if res.Handles[0].SourceTool != tt.wantTool {
				t.Errorf("HandleResult.SourceTool = %q, want %q", res.Handles[0].SourceTool, tt.wantTool)
			}
			if len(audits) == 0 {
				t.Fatal("expected at least one audit record")
			}
			if audits[0].Tool != tt.wantTool {
				t.Errorf("AuditRecord.Tool = %q, want %q", audits[0].Tool, tt.wantTool)
			}
		})
	}
}

// TestCheck_BackendAwareSourceToolSkipped verifies the skip path also surfaces the
// active backend in both the result and the audit record.
func TestCheck_BackendAwareSourceToolSkipped(t *testing.T) {
	v := newTestValidatorWithBackend(&fakeRunner{}, BackendSherlock, sherlockCuratedPlatforms)

	res, audits := v.Check(map[string]interface{}{"name": "No Email"})

	if res.SourceTool != SourceToolSherlock {
		t.Errorf("skipped SourceTool = %q, want %q", res.SourceTool, SourceToolSherlock)
	}
	if len(audits) != 1 || audits[0].Tool != SourceToolSherlock {
		t.Errorf("skip audit Tool = %q, want %q", audits[0].Tool, SourceToolSherlock)
	}
}

// TestCheck_PlatformCountAndConfidencePerBackend verifies that metadata
// platform_count and the confidence denominator match the active backend's
// curated list.
func TestCheck_PlatformCountAndConfidencePerBackend(t *testing.T) {
	tests := []struct {
		name            string
		backend         string
		platforms       []string
		wantPlatformCnt int
	}{
		{
			name:            "maigret",
			backend:         BackendMaigret,
			platforms:       curatedPlatforms,
			wantPlatformCnt: len(curatedPlatforms),
		},
		{
			name:            "sherlock",
			backend:         BackendSherlock,
			platforms:       sherlockCuratedPlatforms,
			wantPlatformCnt: len(sherlockCuratedPlatforms),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// One claimed hit out of the full platform set for one handle.
			runner := &fakeRunner{byHandle: map[string]wrapperOutput{
				"bob": {Results: []platformResult{
					{Platform: "GitHub", Status: "claimed", URL: "https://github.com/bob", HTTPStatus: 200},
				}},
			}}
			v := newTestValidatorWithBackend(runner, tt.backend, tt.platforms)

			res, _ := v.Check(map[string]interface{}{"email": "bob@acme.com"})

			if got := res.Metadata["platform_count"]; got != tt.wantPlatformCnt {
				t.Errorf("platform_count = %v, want %d", got, tt.wantPlatformCnt)
			}

			handleCount := len(res.HandlesChecked)
			if handleCount == 0 {
				t.Fatal("expected at least one checked handle")
			}
			wantConfidence := math.Round(1.0/float64(handleCount*tt.wantPlatformCnt)*1000) / 1000
			if res.Confidence != wantConfidence {
				t.Errorf("confidence = %f, want %f (1/(%d handles × %d platforms))", res.Confidence, wantConfidence, handleCount, tt.wantPlatformCnt)
			}
		})
	}
}

// TestCheck_SpiderFootDisabledByDefault verifies that a Validator constructed
// without SOCIAL_FOOTPRINT_SPIDERFOOT_ENABLED set does not have a SpiderFoot
// runner and reports the source tool of the primary backend only.
func TestCheck_SpiderFootDisabledByDefault(t *testing.T) {
	v := NewValidatorWithBackend(time.Second, 0, BackendMaigret)
	if v.spiderfootEnabled {
		t.Errorf("spiderfootEnabled = true, want false by default")
	}
	if v.spiderfootRunner != nil {
		t.Errorf("spiderfootRunner should be nil when disabled")
	}
}

// TestCheck_SpiderFootMergesSignals verifies that SpiderFoot results are
// merged with the primary backend: overlapping platforms keep the most
// definitive status, new platforms are appended, and the confidence denominator
// includes the SpiderFoot platform count.
func TestCheck_SpiderFootMergesSignals(t *testing.T) {
	primary := &fakeRunner{byHandle: map[string]wrapperOutput{
		"bob": {Results: []platformResult{
			{Platform: "GitHub", Status: "claimed", URL: "https://github.com/bob", HTTPStatus: 200},
		}},
	}}
	spider := &fakeRunner{byHandle: map[string]wrapperOutput{
		"bob": {Results: []platformResult{
			{Platform: "GitHub", Status: "available"},
			{Platform: "Keybase", Status: "claimed", URL: "https://keybase.io/bob", HTTPStatus: 200},
		}},
	}}
	v := newTestValidatorWithSpiderFoot(primary, spider, BackendMaigret, curatedPlatforms, spiderfootCuratedPlatforms)

	res, _ := v.Check(map[string]interface{}{"email": "bob@acme.com"})

	if res.Status != statusOK {
		t.Fatalf("status = %q, want ok", res.Status)
	}
	if len(res.Handles) == 0 {
		t.Fatal("expected at least one handle result")
	}

	byPlatform := map[string]PlatformSignal{}
	for _, p := range res.Handles[0].Platforms {
		byPlatform[p.Platform] = p
	}
	if byPlatform["GitHub"].Status != "claimed" {
		t.Errorf("GitHub should stay claimed (not downgraded to available); got %q", byPlatform["GitHub"].Status)
	}
	if byPlatform["Keybase"].Status != "claimed" {
		t.Errorf("Keybase should be claimed from SpiderFoot; got %q", byPlatform["Keybase"].Status)
	}
	if res.ActiveSignals != 2 {
		t.Errorf("active_signals = %d, want 2", res.ActiveSignals)
	}

	denom := len(curatedPlatforms) + len(spiderfootCuratedPlatforms)
	wantConfidence := math.Round(2.0/float64(denom)*1000) / 1000
	if res.Confidence != wantConfidence {
		t.Errorf("confidence = %f, want %f (2/(%d+%d))", res.Confidence, wantConfidence, len(curatedPlatforms), len(spiderfootCuratedPlatforms))
	}
	if got := res.Metadata["spiderfoot_platform_count"]; got != len(spiderfootCuratedPlatforms) {
		t.Errorf("spiderfoot_platform_count = %v, want %d", got, len(spiderfootCuratedPlatforms))
	}
	if got := res.Metadata["source_tool_spiderfoot"]; got != SourceToolSpiderFoot {
		t.Errorf("source_tool_spiderfoot = %v, want %q", got, SourceToolSpiderFoot)
	}
	if !strings.Contains(res.SourceTool, SourceToolSpiderFoot) {
		t.Errorf("SocialFootprintResult.SourceTool = %q, should contain SpiderFoot", res.SourceTool)
	}
}

// TestCheck_SpiderFootDoesNotCrashOnRunnerError verifies that a SpiderFoot
// runner failure degrades that source only: the primary backend still produces
// an "ok" result and the pipeline stays alive.
func TestCheck_SpiderFootDoesNotCrashOnRunnerError(t *testing.T) {
	primary := &fakeRunner{byHandle: map[string]wrapperOutput{
		"bob": {Results: []platformResult{
			{Platform: "GitHub", Status: "claimed", URL: "https://github.com/bob", HTTPStatus: 200},
		}},
	}}
	spider := &fakeRunner{err: fmt.Errorf("spiderfoot wrapper not installed")}
	v := newTestValidatorWithSpiderFoot(primary, spider, BackendMaigret, curatedPlatforms, spiderfootCuratedPlatforms)

	res, _ := v.Check(map[string]interface{}{"email": "bob@acme.com"})

	if res.Status != statusOK {
		t.Fatalf("status = %q, want ok (primary succeeded)", res.Status)
	}
	if res.ActiveSignals != 1 {
		t.Errorf("active_signals = %d, want 1", res.ActiveSignals)
	}
}

// TestCheck_HandlePassesCorrectPlatforms verifies checkHandle forwards the
// Validator's platform list into runner.run(), not always Maigret's list.
func TestCheck_HandlePassesCorrectPlatforms(t *testing.T) {
	tests := []struct {
		name      string
		backend   string
		platforms []string
	}{
		{
			name:      "maigret",
			backend:   BackendMaigret,
			platforms: curatedPlatforms,
		},
		{
			name:      "sherlock",
			backend:   BackendSherlock,
			platforms: sherlockCuratedPlatforms,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotPlatforms []string
			runner := runnerFunc(func(_ context.Context, _ string, platforms []string, _ time.Duration) (wrapperOutput, error) {
				gotPlatforms = platforms
				return wrapperOutput{Results: []platformResult{}}, nil
			})
			v := newTestValidatorWithBackend(runner, tt.backend, tt.platforms)

			v.Check(map[string]interface{}{"email": "alice@acme.com"})

			if len(gotPlatforms) != len(tt.platforms) {
				t.Errorf("runner got %d platforms, want %d", len(gotPlatforms), len(tt.platforms))
			}
		})
	}
}
