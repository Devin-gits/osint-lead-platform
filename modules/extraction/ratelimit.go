package extraction

import (
	"sync"
	"time"
)

// rateLimiter enforces a minimum spacing between consecutive Extract calls on a
// single Extractor. When a caller loops over many URLs reusing one Extractor,
// each call blocks until the effective interval has elapsed since the previous
// one. The effective interval starts at minInterval and doubles on consecutive
// errors up to maxInterval, then resets on a successful extraction.
//
// It is intentionally simple and in-process (no external dependency). The first
// call on a fresh limiter is never delayed.
type rateLimiter struct {
	mu          sync.Mutex
	minInterval time.Duration
	maxInterval time.Duration
	current     time.Duration
	last        time.Time
	sleep       func(time.Duration)
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

func (r *rateLimiter) interval() time.Duration {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.current == 0 {
		return r.minInterval
	}
	return r.current
}

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

func (r *rateLimiter) backoff() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.current *= 2
	if r.maxInterval > 0 && r.current > r.maxInterval {
		r.current = r.maxInterval
	}
}

func (r *rateLimiter) reset() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.current = r.minInterval
}
