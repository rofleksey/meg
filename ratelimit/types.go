package ratelimit

import (
	"sync"
	"time"
)

// Limiter is a bounded per-key token-bucket limiter. It is deliberately local
// to one process: it protects the public HTTP boundary without making
// availability depend on a shared cache. Deployments that run more than one
// control-plane replica should also apply an equivalent edge rate limit.
type Limiter struct {
	mu      sync.Mutex
	max     int
	window  time.Duration
	now     func() time.Time
	entries map[string]limiterEntry
}
