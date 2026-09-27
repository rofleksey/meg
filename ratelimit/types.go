package ratelimit

import (
	"sync"
	"time"
)

// Limiter is a bounded per-key token-bucket limiter local to one process.
// Applications with multiple replicas must enforce a shared or upstream
// budget when they need a global limit.
type Limiter struct {
	mu      sync.Mutex
	max     int
	window  time.Duration
	now     func() time.Time
	entries map[string]limiterEntry
}
