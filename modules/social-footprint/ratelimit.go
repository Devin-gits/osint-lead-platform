package socialfootprint

import (
	"sync"
	"time"
)

// rateLimiter enforces a minimum spacing between consecutive per-lead checks on
// a single Validator. This is the code-level guardrail behind the decision doc's
// requirement that social-footprint be a "rate-limited, per-lead spot check,
// never a bulk sweep": when a caller loops over many leads reusing one Validator,
// each Check blocks until the effective interval has elapsed since the previous
// one. The effective interval starts at minInterval and doubles on consecutive
// errors up to maxInterval, then resets on a successful lead.
//
// It is intentionally simple and in-process (no external dependency). The first
// call on a fresh limiter is never delayed (last is zero).
type rateLimiter struct {
	mu          sync.Mutex
	minInterval time.Duration
	maxInterval time.Duration
	current     time.Duration
	last        time.Time
	// sleep is indirected so tests can assert on the enforced delay without
	// actually sleeping in real time.
	sleep func(time.Duration)
}

func newRateLimiter(minInterval time.Duration) *rateLimiter {
	maxInterval := 60 * time.Second
	if minInterval > 0 && minInterval*5 < maxInterval {
		maxInterval = minInterval * 5
	}
	return &rateLimiter{
		minInterval: minInterval,
		maxInterval: maxInterval,
		current:     minInterval,
		sleep:       time.Sleep,
	}
}

// interval returns the currently effective wait interval (useful for tests
// and for displaying the active rate-limit state).
func (r *rateLimiter) interval() time.Duration {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.current == 0 {
		return r.minInterval
	}
	return r.current
}

// wait blocks until the effective interval has elapsed since the previous wait
// returned, then records the new invocation time.
func (r *rateLimiter) wait() {
	r.mu.Lock()
	defer r.mu.Unlock()

	if !r.last.IsZero() {
		wait := r.current
		if wait <= 0 {
			wait = r.minInterval
		}
		if wait > 0 {
			elapsed := time.Since(r.last)
			if elapsed < wait {
				r.sleep(wait - elapsed)
			}
		}
	}
	r.last = time.Now()
}

// backoff doubles the current effective interval, capped at maxInterval, so
// repeated consecutive failures do not hammer a site or the wrapper.
func (r *rateLimiter) backoff() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.current *= 2
	if r.maxInterval > 0 && r.current > r.maxInterval {
		r.current = r.maxInterval
	}
}

// reset restores the effective interval to the configured minimum.
func (r *rateLimiter) reset() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.current = r.minInterval
}
