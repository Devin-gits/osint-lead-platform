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

func testLimiter(minInterval time.Duration, slept *time.Duration) *rateLimiter {
	r := newRateLimiter(minInterval)
	r.sleep = func(d time.Duration) { *slept += d }
	return r
}

// TestRateLimiter_FirstCallNoDelay verifies the first wait on a fresh limiter is
// never delayed (so a single-lead CLI run does not sleep).
func TestRateLimiter_FirstCallNoDelay(t *testing.T) {
	var slept time.Duration
	r := testLimiter(5*time.Second, &slept)
	r.wait()
	if slept != 0 {
		t.Errorf("first call slept %s, want 0", slept)
	}
}

// TestRateLimiter_SecondCallSpaced verifies the second consecutive wait is
// delayed by ~minInterval, enforcing per-lead spacing (the scope guardrail).
func TestRateLimiter_SecondCallSpaced(t *testing.T) {
	var slept time.Duration
	r := testLimiter(5*time.Second, &slept)
	r.wait() // first: no delay, sets last
	r.wait() // second: immediately after, should request a delay close to 5s
	if slept <= 0 || slept > 5*time.Second {
		t.Errorf("second call slept %s, want (0, 5s]", slept)
	}
}

// TestRateLimiter_ZeroIntervalDisabled verifies a zero interval disables spacing.
func TestRateLimiter_ZeroIntervalDisabled(t *testing.T) {
	var slept time.Duration
	r := testLimiter(0, &slept)
	r.wait()
	r.wait()
	if slept != 0 {
		t.Errorf("zero-interval limiter slept %s, want 0", slept)
	}
}

// TestRateLimiter_Backoff verifies that consecutive failures double the
// effective interval up to the configured cap.
func TestRateLimiter_Backoff(t *testing.T) {
	r := newRateLimiter(5 * time.Second)
	if r.interval() != 5*time.Second {
		t.Fatalf("initial interval = %s, want 5s", r.interval())
	}
	r.backoff()
	if r.interval() != 10*time.Second {
		t.Errorf("after 1 backoff interval = %s, want 10s", r.interval())
	}
	r.backoff()
	r.backoff()
	r.backoff()
	r.backoff() // 5 -> 10 -> 20 -> 40 -> 80 (capped at 25s for min=5*5)
	if r.interval() != 25*time.Second {
		t.Errorf("capped interval = %s, want 25s", r.interval())
	}
}

// TestRateLimiter_Reset verifies a successful lead restores the effective
// interval to the configured minimum.
func TestRateLimiter_Reset(t *testing.T) {
	r := newRateLimiter(5 * time.Second)
	r.backoff()
	r.backoff()
	if r.interval() == 5*time.Second {
		t.Fatal("expected backoff to increase interval before reset")
	}
	r.reset()
	if r.interval() != 5*time.Second {
		t.Errorf("after reset interval = %s, want 5s", r.interval())
	}
}
