package ratelimit

import (
	"time"

	"golang.org/x/time/rate"
)

const (
	maxTrackedKeys = 4096
	unknownKey     = "unknown"
)

type limiterEntry struct {
	bucket   *rate.Limiter
	lastSeen time.Time
}

func normalizeKey(key string) string {
	if key == "" {
		return unknownKey
	}

	return key
}

func (l *Limiter) entry(key string, now time.Time) limiterEntry {
	if current, ok := l.entries[key]; ok {
		current.lastSeen = now
		l.entries[key] = current

		return current
	}

	if len(l.entries) >= maxTrackedKeys {
		l.evictOldest()
	}

	current := limiterEntry{
		bucket:   rate.NewLimiter(rate.Limit(float64(l.max)/l.window.Seconds()), l.max),
		lastSeen: now,
	}
	l.entries[key] = current

	return current
}

func (l *Limiter) evictOldest() {
	var (
		oldestKey string
		oldest    time.Time
	)
	for key, current := range l.entries {
		if oldest.IsZero() || current.lastSeen.Before(oldest) {
			oldestKey = key
			oldest = current.lastSeen
		}
	}

	delete(l.entries, oldestKey)
}
