// Package security contains HTTP security primitives.
package ratelimit

import "time"

// NewLimiter creates a limiter with one independent budget for every key.
func NewLimiter(limit int, window time.Duration) *Limiter {
	return &Limiter{
		max:     limit,
		window:  window,
		now:     time.Now,
		entries: make(map[string]limiterEntry),
	}
}

// Allow consumes one request from key's budget. retryAfter is non-zero only
// when the request is rejected.
func (l *Limiter) Allow(key string) (allowed bool, retryAfter time.Duration) {
	if l == nil || l.max <= 0 || l.window <= 0 {
		return true, 0
	}

	now := l.now()
	l.mu.Lock()
	defer l.mu.Unlock()

	current := l.entry(normalizeKey(key), now)

	reservation := current.bucket.ReserveN(now, 1)
	if !reservation.OK() {
		return false, l.window
	}

	retryAfter = reservation.DelayFrom(now)
	if retryAfter > 0 {
		reservation.CancelAt(now)

		return false, retryAfter
	}

	return true, 0
}
