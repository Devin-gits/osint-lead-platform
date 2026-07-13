package socialfootprint

import (
	"sync"
	"time"
)

// rateLimiter enforces a minimum spacing between consecutive per-lead checks on
// a single Validator. This is the code-level guardrail behind the decision doc's
// requirement that social-footprint be a "rate-limited, per-lead spot check,
// never a bulk sweep": when a caller loops over many leads reusing one Validator,
// each Check blocks until minInterval has elapsed since the previous one.
//
// It is intentionally simple and in-process (no external dependency). The first
// call on a fresh limiter is never delayed (last is zero).
type rateLimiter struct {
	mu          sync.Mutex
	minInterval time.Duration
	last        time.Time
	// sleep is indirected so tests can assert on the enforced delay without
	// actually sleeping in real time.
	sleep func(time.Duration)
}

func newRateLimiter(minInterval time.Duration) *rateLimiter {
	return &rateLimiter{minInterval: minInterval, sleep: time.Sleep}
}

// wait blocks until at least minInterval has elapsed since the previous wait
// returned, then records the new invocation time.
func (r *rateLimiter) wait() {
	r.mu.Lock()
	defer r.mu.Unlock()

	if !r.last.IsZero() && r.minInterval > 0 {
		elapsed := time.Since(r.last)
		if elapsed < r.minInterval {
			r.sleep(r.minInterval - elapsed)
		}
	}
	r.last = time.Now()
}
