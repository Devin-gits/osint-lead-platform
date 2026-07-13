package socialfootprint

import (
	"testing"
	"time"
)

func assertContains(t *testing.T, hay []string, needle string) {
	t.Helper()
	for _, h := range hay {
		if h == needle {
			return
		}
	}
	t.Errorf("expected %v to contain %q", hay, needle)
}

// TestRateLimiter_FirstCallNoDelay verifies the first wait on a fresh limiter is
// never delayed (so a single-lead CLI run does not sleep).
func TestRateLimiter_FirstCallNoDelay(t *testing.T) {
	var slept time.Duration
	r := &rateLimiter{minInterval: 5 * time.Second, sleep: func(d time.Duration) { slept += d }}
	r.wait()
	if slept != 0 {
		t.Errorf("first call slept %s, want 0", slept)
	}
}

// TestRateLimiter_SecondCallSpaced verifies the second consecutive wait is
// delayed by ~minInterval, enforcing per-lead spacing (the scope guardrail).
func TestRateLimiter_SecondCallSpaced(t *testing.T) {
	var slept time.Duration
	r := &rateLimiter{minInterval: 5 * time.Second, sleep: func(d time.Duration) { slept += d }}
	r.wait() // first: no delay, sets last
	r.wait() // second: immediately after, should request a delay close to 5s
	if slept <= 0 || slept > 5*time.Second {
		t.Errorf("second call slept %s, want (0, 5s]", slept)
	}
}

// TestRateLimiter_ZeroIntervalDisabled verifies a zero interval disables spacing.
func TestRateLimiter_ZeroIntervalDisabled(t *testing.T) {
	var slept time.Duration
	r := &rateLimiter{minInterval: 0, sleep: func(d time.Duration) { slept += d }}
	r.wait()
	r.wait()
	if slept != 0 {
		t.Errorf("zero-interval limiter slept %s, want 0", slept)
	}
}
