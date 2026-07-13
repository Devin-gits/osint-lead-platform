package socialfootprint

import (
	"context"
	"errors"
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
	return &Validator{timeout: time.Second, limiter: newRateLimiter(0), runner: r}
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
